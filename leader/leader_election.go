package leader

import (
    log "github.com/sirupsen/logrus"
    "github.com/sobitada/go-cardano"
    "github.com/sobitada/go-jormungandr/api"
    "github.com/sobitada/thor/monitor"
    "github.com/sobitada/thor/utils"
    "math/big"
    "math/rand"
    "strings"
    "sync"
    "time"
)

type Jury struct {
    nodes            map[string]monitor.Node
    nodeStatsChannel chan map[string]api.NodeStatistic

    watchDog        *monitor.ScheduleWatchDog
    scheduleChannel chan []api.LeaderAssignment

    leader      *currentLeader
    leaderMutex *sync.Mutex
    cert        api.LeaderCertificate

    settings JurySettings
}

type currentLeader struct {
    name     string
    leaderID uint64
}

// settings for the Leader Jury.
type JurySettings struct {
    // the number of checkpoints that shall be
    // considered for the health check and leader
    // decisions.
    Window int
    // the time window in front of a scheduled
    // block in which no leader change is allowed.
    ExclusionZone time.Duration
    // specifies the number of slots before an epoch
    // turn over in which no leader change is allowed.
    PreEpochTurnOverExclusionSlots *big.Int
    // time settings for the block chain such as
    // creation time of the genesis block, slots
    // per epoch and slot duration.
    TimeSettings *cardano.TimeSettings
}

// gets the leader jury judging the given nodes. it expects the certificate of the
// leader that shall be managed and jury settings. moreover, the time
// settings for the block chain is needed to handle epoch turn overs.
func GetLeaderJuryFor(nodes []monitor.Node, mon *monitor.NodeMonitor, watchDog *monitor.ScheduleWatchDog,
    certificate api.LeaderCertificate, settings JurySettings) (*Jury, error) {
    // create a node map
    nodeMap := make(map[string]monitor.Node)
    for i := range nodes {
        currentNode := nodes[i]
        if currentNode.Type == monitor.LeaderCandidate {
            nodeMap[nodes[i].Name] = currentNode
        }
    }
    if len(nodeMap) == 0 {
        log.Warnf("No node has been specified as leader candidate.")
        return nil, nil
    }
    // register the node statistics listener
    if mon == nil {
        return nil, invalidArgument{
            Method: "GetLeaderJuryFor",
            Reason: "The passed node monitor must not be nil.",
        }
    }
    nodeStatsChannel := make(chan map[string]api.NodeStatistic)
    mon.ListenerManager.RegisterNodeStatisticListener(nodeStatsChannel)
    // register schedule listener
    if watchDog == nil {
        return nil, invalidArgument{
            Method: "GetLeaderJuryFor",
            Reason: "The passed watchdog must not be nil.",
        }
    }
    scheduleChannel := make(chan []api.LeaderAssignment)
    watchDog.RegisterListener(scheduleChannel)
    return &Jury{
        nodes:            nodeMap,
        nodeStatsChannel: nodeStatsChannel,
        watchDog:         watchDog,
        scheduleChannel:  scheduleChannel,
        cert:             certificate,
        settings:         settings,
        leaderMutex:      &sync.Mutex{},
    }, nil
}

// scans for the current leader among all the nodes,
// it expects only one leader node. multiple pool
// certificates are not supported. This scanner also
// tries to correct potential problems with the current
// nodes. There must not be multiple nodes promoted to
// leader.
func (jury *Jury) scanForLeader() *currentLeader {
    var leader *currentLeader = nil
    for name, node := range jury.nodes {
        leaderIDs, err := node.API.GetRegisteredLeaders()
        if err == nil {
            if len(leaderIDs) > 0 && leader == nil {
                leader = &currentLeader{name: name, leaderID: leaderIDs[0]}
                if len(leaderIDs) > 1 {
                    otherLeaderIDs := leaderIDs[1:]
                    for i := range otherLeaderIDs {
                        demoteLeader(node, otherLeaderIDs[i], 3)
                    }
                }
            } else if len(leaderIDs) > 0 {
                for i := range leaderIDs {
                    demoteLeader(node, leaderIDs[i], 3)
                }
            }
        }
    }
    return leader
}

// takes the given old map and only returns the entries of the map
// in a new map that are associated with a key in the given
// list of leaders, i.e. it filters all
func mapWithViableLeaders(leaders []string, oldMap map[string]*big.Float) map[string]*big.Float {
    newMap := make(map[string]*big.Float)
    for _, leader := range leaders {
        value, found := oldMap[leader]
        if found {
            newMap[leader] = value
        }
    }
    return newMap
}

// starts the leader jury and let it continuously run. it reads all the checkpoints
// that have been passed from the monitor to this leader jury.
func (jury *Jury) Judge() {
    nodeNames := monitor.GetNodeNames(jury.nodes)
    mem := createBlockHeightMemory(nodeNames, jury.settings.Window)
    // get current leader
    leader := jury.scanForLeader()
    if leader != nil {
        log.Infof("[LEADER JURY] Node %v is elected and has ID=%v", leader.name, leader.leaderID)
        jury.leader = leader
    }
    // start sanity management
    go jury.startSanityChecks()
    go jury.turnOverHandling()
    // turn over preparation
    for ; ; {
        latestBlockStats := <-jury.nodeStatsChannel
        // check the leader schedule
        currentSlotDate, _ := jury.settings.TimeSettings.GetSlotDateFor(time.Now())
        schedule, found := jury.watchDog.GetScheduleFor(currentSlotDate.GetEpoch())
        if !found || schedule == nil {
            schedule = []api.LeaderAssignment{}
        }
        // check health
        viableNodeNames := jury.watchDog.GetViableLeaderNodes()
        log.Infof("[LEADER JURY] Viable Nodes are [%v].", strings.Join(viableNodeNames, ","))
        mem.addBlockHeights(latestBlockStats)
        if len(viableNodeNames) > 0 {
            maxConf, maxConfNodes := utils.MinFloat(mapWithViableLeaders(viableNodeNames, mem.computeHealth()))
            log.Infof("[LEADER JURY] Nodes [%v] have lowest drift (%v).", strings.Join(maxConfNodes, ","), maxConf)
            _, bestLCNodes := utils.MaxFloat(mapUpTime(maxConfNodes, latestBlockStats))
            log.Infof("[LEADER JURY] Nodes [%v] considered to be healthiest.", strings.Join(bestLCNodes, ","))
            if bestLCNodes != nil && len(bestLCNodes) > 0 {
                if jury.leader == nil || !containsLeader(bestLCNodes, jury.leader.name) {
                    if len(bestLCNodes) > 0 {
                        // no leader change if in exclusion zone.
                        if len(schedule) > 0 {
                            futureSchedule := api.FilterLeaderLogsBefore(time.Now().Add(-2*jury.settings.TimeSettings.SlotDuration), schedule)
                            if len(futureSchedule) > 0 {
                                timeToNextBlock := futureSchedule[0].ScheduleTime.Sub(time.Now())
                                if timeToNextBlock < jury.settings.ExclusionZone {
                                    log.Warnf("[LEADER JURY] In exclusion zone before scheduled block.")
                                    continue
                                }
                            }
                        }
                        // no leader change in exclusion zone before epoch turn over.
                        if new(big.Int).Sub(jury.settings.TimeSettings.SlotsPerEpoch,
                            currentSlotDate.GetSlot()).Cmp(jury.settings.PreEpochTurnOverExclusionSlots) <= 0 {
                            log.Warnf("[LEADER JURY] In exclusion zone before epoch turn over, no leader change will be performed.")
                            continue
                        }
                        // change leader.
                        jury.changeLeader(randomSort(bestLCNodes)[0])
                    }
                }
            }
        }
        if jury.leader != nil {
            log.Infof("[LEADER JURY] Current Leader is %v.", jury.leader.name)
        }
    }
}

// maps the uptime to the node name.
func mapUpTime(nodeNames []string, latestBlockStats map[string]api.NodeStatistic) map[string]*big.Float {
    uptimeMap := make(map[string]*big.Float)
    for n := range nodeNames {
        name := nodeNames[n]
        uptimeMap[name] = new(big.Float).SetInt64(int64(latestBlockStats[name].UpTime))
    }
    return uptimeMap
}

// returns a randomly sorted list of nodes.
func randomSort(nodes []string) []string {
    rand.Seed(time.Now().UnixNano())
    rand.Shuffle(len(nodes), func(i, j int) { nodes[i], nodes[j] = nodes[j], nodes[i] })
    return nodes
}

// changes the leader to the given name.
func (jury *Jury) changeLeader(leaderName string) {
    jury.leaderMutex.Lock()
    defer jury.leaderMutex.Unlock()

    newLeaderNode := jury.nodes[leaderName]
    leaderID, err := newLeaderNode.API.PostLeader(jury.cert)
    if err == nil {
        if jury.leader != nil {
            go demoteLeader(jury.nodes[jury.leader.name], jury.leader.leaderID, 3)
        }
        jury.leader = &currentLeader{name: newLeaderNode.Name, leaderID: leaderID}
        log.Infof("[LEADER JURY] Node %v is elected and has ID=%v", newLeaderNode.Name, leaderID)
    } else {
        log.Errorf("[LEADER JURY] Could not change to leader %v. %v", newLeaderNode.Name, err.Error())
    }
}

// tries at first in n attempts to demote the given leader node. if this fails,
// then the leader node is shut down as a safety measure.
func demoteLeader(node monitor.Node, ID uint64, attempts int) {
    demoted := false
    for i := 0; i < attempts; i++ {
        found, err := node.API.RemoveRegisteredLeader(ID)
        if err != nil {
            log.Warnf("[LEADER JURY] The leader node %v could not be demoted. Attempt: %v. %v. ", node.Name, i+1, err.Error())
            time.Sleep(1 * time.Second)
        } else if !found {
            log.Warnf("[LEADER JURY] The node %v was not in leader mode.", node.Name)
            demoted = true
            break
        } else {
            demoted = true
            break
        }
    }
    if !demoted {
        log.Warnf("[LEADER JURY] Could not demote %v. Now a shutdown will be tried.", node.Name)
        monitor.ShutDownNode(node)
    }
}

// checks whether the given list of nodes contains the leader.
func containsLeader(nodes []string, leader string) bool {
    for _, a := range nodes {
        if a == leader {
            return true
        }
    }
    return false
}
