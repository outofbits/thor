package monitor

import (
    log "github.com/sirupsen/logrus"
    "github.com/sobitada/go-cardano"
    "github.com/sobitada/go-jormungandr/api"
    "github.com/sobitada/thor/utils"
    "math/big"
    "math/rand"
    "strings"
    "sync"
    "time"
)

type Jury struct {
    leader            *currentLeader
    leaderMutex       *sync.Mutex
    nodes             map[string]Node
    BlockStatsChannel chan map[string]api.NodeStatistic
    Cert              api.LeaderCertificate
    settings          JurySettings
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
    // specifies the number of slots after an epoch
    // turn over in which no leader change is allowed.
    EpochTurnOverExclusionSlots *big.Int
    // time settings for the block chain such as
    // creation time of the genesis block, slots
    // per epoch and slot duration.
    TimeSettings *cardano.TimeSettings
}

// gets the leader jury judging the given nodes. it expects the certificate of the
// leader that shall be managed and jury settings. moreover, the time
// settings for the block chain is needed to handle epoch turn overs.
func GetLeaderJuryFor(nodes []Node, certificate api.LeaderCertificate, settings JurySettings) *Jury {
    c := make(chan map[string]api.NodeStatistic)
    nodeMap := make(map[string]Node)
    for i := range nodes {
        nodeMap[nodes[i].Name] = nodes[i]
    }
    var leaderRWMutex sync.Mutex
    return &Jury{
        nodes:             nodeMap,
        BlockStatsChannel: c,
        Cert:              certificate,
        settings:          settings,
        leaderMutex:       &leaderRWMutex,
    }
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

// gets the current schedule.
func getCurrentSchedule(epoch *big.Int, node Node) []api.LeaderAssignment {
    schedule, err := node.API.GetLeadersSchedule()
    if err == nil && schedule != nil {
        return api.FilterForLeaderLogsInEpoch(epoch, api.SortLeaderLogsByScheduleTime(schedule))
    }
    return schedule
}

// the current strategy to handle epoch turn overs is to stick to the
// current leader for a while, and then bootstrap one after another of
// the other passive nodes.
func shutdownTrustedNodesGracefully(leaderName *string, nodes map[string]Node) {
    for name, node := range nodes {
        if leaderName == nil || name != *leaderName {
            log.Debugf("Node %v is going to be shutdown gracefully.", name)
            ShutDownNode(node)
            time.Sleep(1 * time.Minute)
        }
    }
}

// starts the leader jury and let it continuously run. it reads all the checkpoints
// that have been passed from the monitor to this leader jury.
func (jury *Jury) Judge() {
    nodeNames := GetNodeNames(jury.nodes)
    mem := createBlockHeightMemory(nodeNames, jury.settings.Window)
    // get current leader
    leader := jury.scanForLeader()
    if leader != nil {
        log.Infof("[LEADER JURY] Node %v is elected and has ID=%v", leader.name, leader.leaderID)
        jury.leader = leader
    }
    // schedule management preparation
    var scheduleMap = make(map[string][]api.LeaderAssignment)
    var lastScheduleCheckMap = make(map[string]time.Time)
    scheduleChannel := make(chan []api.LeaderAssignment)
    go jury.sanityCheck(scheduleChannel)
    // turn over preparation
    turnOverBootStrap := make(map[string]bool)
    for ; ; {
        latestBlockStats := <-jury.BlockStatsChannel
        // check the leader schedule
        currentSlotDate, _ := jury.settings.TimeSettings.GetSlotDateFor(time.Now())
        schedule, found := scheduleMap[currentSlotDate.GetEpoch().String()]
        if !found || len(schedule) == 0 {
            lastTimeChecked, found := lastScheduleCheckMap[currentSlotDate.GetEpoch().String()]
            if !found || (lastTimeChecked.Before(time.Now().Add(-10 * time.Minute))) {
                // wait two minutes after epoch turn over.
                if currentSlotDate.GetSlot().Cmp(jury.settings.EpochTurnOverExclusionSlots) < 0 {
                    time.Sleep(2 * time.Minute)
                }
                // fetch the assignment schedule
                var newSchedule []api.LeaderAssignment
                if jury.leader != nil {
                    newSchedule = getCurrentSchedule(currentSlotDate.GetEpoch(), jury.nodes[jury.leader.name])
                }
                if newSchedule == nil || len(newSchedule) > 0 {
                    for n := range jury.nodes {
                        if jury.leader != nil && jury.leader.name == jury.nodes[n].Name {
                            continue
                        }
                        newSchedule = getCurrentSchedule(currentSlotDate.GetEpoch(), jury.nodes[n])
                        if newSchedule != nil && len(newSchedule) > 0 {
                            break
                        }
                    }
                }
                if newSchedule != nil && len(newSchedule) > 0 {
                    scheduleChannel <- newSchedule
                }
                schedule = newSchedule
                scheduleMap[currentSlotDate.GetEpoch().String()] = newSchedule
                lastScheduleCheckMap[currentSlotDate.GetEpoch().String()] = time.Now()
            }
        }
        if schedule == nil {
            schedule = []api.LeaderAssignment{}
        }
        log.Debugf("Number of leader assignments: %v", len(schedule))
        // bootstrap non leader nodes after epoch turn over gracefully. //TODO: make stateless
        if currentSlotDate.GetSlot().Cmp(jury.settings.EpochTurnOverExclusionSlots) < 0 {
            bootstrapped, found := turnOverBootStrap[currentSlotDate.GetEpoch().String()]
            if !found || !bootstrapped {
                log.Debugf("[LEADER JURY] Entry into exclusion zone, none leader nodes are shutdown.")
                if jury.leader != nil {
                    go shutdownTrustedNodesGracefully(&jury.leader.name, jury.nodes)
                } else {
                    go shutdownTrustedNodesGracefully(nil, jury.nodes)
                }
                turnOverBootStrap[currentSlotDate.GetEpoch().String()] = true
            }
        }
        // check health
        mem.addBlockHeights(latestBlockStats)
        maxConf, maxConfNodes := utils.MinFloat(mem.computeHealth())
        log.Infof("[LEADER JURY] Nodes [%v] have lowest drift (%v).", strings.Join(maxConfNodes, ","), maxConf)
        if jury.leader == nil || !containsLeader(maxConfNodes, jury.leader.name) {
            if len(maxConfNodes) > 0 {
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
                // no leader change after in exclusion zone after epoch turn over.
                if currentSlotDate.GetSlot().Cmp(jury.settings.EpochTurnOverExclusionSlots) < 0 {
                    log.Warnf("[LEADER JURY] In exclusion zone, no leader change will be performed.")
                    continue
                }
                // change leader.
                jury.changeLeader(randomSort(maxConfNodes)[0])
            }
        }
        if jury.leader != nil {
            log.Infof("[LEADER JURY] Current Leader is %v.", jury.leader.name)
        }
    }
}

// returns a randomly sorted list of nodes.
func randomSort(nodes []string) []string {
    list := nodes[:]
    rand.Seed(time.Now().UnixNano())
    rand.Shuffle(len(list), func(i, j int) { list[i], list[j] = list[j], list[i] })
    return list
}

// changes the leader to the given name.
func (jury *Jury) changeLeader(leaderName string) {
    jury.leaderMutex.Lock()
    defer jury.leaderMutex.Unlock()

    newLeaderNode := jury.nodes[leaderName]
    leaderID, err := newLeaderNode.API.PostLeader(jury.Cert)
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
func demoteLeader(node Node, ID uint64, attempts int) {
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
        ShutDownNode(node)
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
