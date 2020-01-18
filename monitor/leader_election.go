package monitor

import (
    "fmt"
    log "github.com/sirupsen/logrus"
    "github.com/sobitada/go-cardano"
    "github.com/sobitada/go-jormungandr/api"
    "math/big"
    "math/rand"
    "strings"
    "sync"
    "time"
)

type LeaderJury struct {
    leader            *currentLeader
    leaderMutex       *sync.Mutex
    nodes             map[string]Node
    BlockStatsChannel chan map[string]api.NodeStatistic
    Cert              api.LeaderCertificate
    config            LeaderJuryConfig
}

type currentLeader struct {
    name     string
    leaderID uint64
}

// configuration details for the Leader Jury.
type LeaderJuryConfig struct {
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
    // time settings for the blockchain such as
    // creation time of the genesis block, slots
    // per epoch and slot duration.
    TimeSettings *cardano.TimeSettings
}

// gets the leader jury judging the given nodes. it expects the certificate of the
// leader that shall be managed and configuration details. moreover, the time
// settings for the blockhain is needed to handle epoch turn overs.
func GetLeaderJuryFor(nodes []Node, certificate api.LeaderCertificate, config LeaderJuryConfig) *LeaderJury {
    c := make(chan map[string]api.NodeStatistic)
    nodeMap := make(map[string]Node)
    for i := range nodes {
        nodeMap[nodes[i].Name] = nodes[i]
    }
    var leaderRWMutex sync.Mutex
    return &LeaderJury{
        nodes:             nodeMap,
        BlockStatsChannel: c,
        Cert:              certificate,
        config:            config,
        leaderMutex:       &leaderRWMutex,
    }
}

// scans for the current leader among all the nodes,
// it expects only one leader node. multiple pool
// certificates are not supported. This scanner also
// tries to correct potential problems with the current
// nodes. There must not be multiple nodes promoted to
// leader.
func (jury *LeaderJury) scanForLeader() *currentLeader {
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

func (jury *LeaderJury) sanityCheckPassiveNode(node Node) {
    leaderIDs, err := node.API.GetRegisteredLeaders()
    if err == nil {
        if len(leaderIDs) > 0 {
            log.Warnf("[LEADER JURY] Node %v is in leader mode while jury promoted other node.", node.Name)
            for i := range leaderIDs {
                demoteLeader(node, leaderIDs[i], 3)
            }
        }
    }
}

func (jury *LeaderJury) sanityCheckLeaderNode(node Node) {
    leaderIDs, err := node.API.GetRegisteredLeaders()
    if err == nil {
        leaderIDNumber := len(leaderIDs)
        if leaderIDNumber == 0 {
            log.Warnf("[LEADER JURY] Node %v is not promoted to leader node as expected.", node.Name)
            leaderID, err := node.API.PostLeader(jury.Cert)
            if err == nil {
                jury.leader = &currentLeader{name: node.Name, leaderID: leaderID}
                log.Infof("[LEADER JURY] Node %v is elected and has ID=%v", node.Name, leaderID)
            } else {
                log.Errorf("[LEADER JURY] Could not change to leader %v. %v", node.Name, err.Error())
            }
        } else if leaderIDNumber == 1 {
            log.Infof("[LEADER JURY] Node %v is leader as expected.", node.Name)
        } else {
            log.Warnf("[LEADER JURY] Node %v has more than one leader registered (%v).", node.Name, leaderIDNumber)
            for i := range leaderIDs {
                if leaderIDs[i] != jury.leader.leaderID {
                    demoteLeader(node, leaderIDs[i], 3)
                }
            }
        }
    }
}

// sanity check tries to correct fail overs and potential flaws in this
// program as well as in Jormungandr. It checks whether only one node is
// promoted to a leader. It is important to avoid adversarial forks,
// because creating such a fork causes public shame! shame! shaming and
// blacklisting.
func (jury *LeaderJury) sanityCheck(scheduleChannel chan []api.LeaderAssignment) {
    for ; ; {
        assignments := <-scheduleChannel
        currentSlotDate, err := jury.config.TimeSettings.GetSlotDateFor(time.Now())
        if err != nil {
            log.Fatalf("[LEADER JURY] Sanity check loop panicked: %v", err.Error())
            time.Sleep(30 * time.Minute)
            continue
        }
        nextAssignments := api.FilterLeaderLogsBefore(time.Now().Add(2*time.Minute),
            api.SortLeaderLogsByScheduleTime(api.FilterForLeaderLogsInEpoch(currentSlotDate.GetEpoch(), assignments)))
        log.Debugf("[LEADER JURY] Started sanity check for %v assignments ahead. ", len(nextAssignments))
        for i := 0; i < len(nextAssignments); i++ {
            waitDuration := nextAssignments[i].ScheduleTime.Sub(time.Now()) - 1*time.Minute
            if waitDuration > 0 { // no sanity check between slots that are too close to each other.
                log.Infof("[LEADER JURY] Waiting %v for the next sanity check.", waitDuration.String())
                time.Sleep(waitDuration)
                log.Infof("[LEADER JURY] Sanity check before assignment %v.", nextAssignments[i].ScheduleTime)
                // do sanity checking
                jury.leaderMutex.Lock()
                for name, node := range jury.nodes {
                    log.Infof("[LEADER JURY] Sanity check node %v.", name)
                    if jury.leader != nil && jury.leader.name == name {
                        jury.sanityCheckLeaderNode(node)
                    } else {
                        jury.sanityCheckPassiveNode(node)
                    }
                }
                jury.leaderMutex.Unlock()
            }
        }
        time.Sleep(1 * time.Minute)
    }
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
            log.Debugf("Node %v is going to be shutdown gracefully.", leaderName)
            shutDownNode(node)
            time.Sleep(1 * time.Minute)
        }
    }
}

// starts the leader jury and let it continuously run. it reads all the checkpoints
// that have been passed from the monitor to this leader jury.
func (jury *LeaderJury) Judge() {
    nodeNames := getNodeNames(jury.nodes)
    mem := createBlockHeightMemory(nodeNames, jury.config.Window)
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
        currentSlotDate, _ := jury.config.TimeSettings.GetSlotDateFor(time.Now())
        schedule, found := scheduleMap[currentSlotDate.GetEpoch().String()]
        if !found || len(schedule) == 0 {
            lastTimeChecked, found := lastScheduleCheckMap[currentSlotDate.GetEpoch().String()]
            if !found || (lastTimeChecked.Before(time.Now().Add(-10 * time.Minute))) {
                // wait two minutes after epoch turn over.
                if currentSlotDate.GetSlot().Cmp(jury.config.EpochTurnOverExclusionSlots) < 0 {
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
                    schedule = newSchedule
                    scheduleMap[currentSlotDate.GetEpoch().String()] = newSchedule
                }
                lastScheduleCheckMap[currentSlotDate.GetEpoch().String()] = time.Now()
            }
        }
        if schedule == nil {
            schedule = []api.LeaderAssignment{}
        }
        log.Debugf("Number of leader assignments: %v", len(schedule))
        // bootstrap non leader nodes after epoch turn over gracefully. //TODO: make stateless
        if currentSlotDate.GetSlot().Cmp(jury.config.EpochTurnOverExclusionSlots) < 0 {
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
        maxConf, maxConfNodes := minFloat(computeHealth(mem.getDiff()))
        log.Infof("[LEADER JURY] Nodes [%v] have lowest drift (%v).", strings.Join(maxConfNodes, ","), maxConf)
        if jury.leader == nil || !containsLeader(maxConfNodes, jury.leader.name) {
            if len(maxConfNodes) > 0 {
                // no leader change if in exclusion zone.
                if len(schedule) > 0 {
                    futureSchedule := api.FilterLeaderLogsBefore(time.Now(), schedule)
                    if len(futureSchedule) > 0 {
                        timeToNextBlock := futureSchedule[0].ScheduleTime.Sub(time.Now())
                        if timeToNextBlock < jury.config.ExclusionZone {
                            log.Warnf("[LEADER JURY] In exclusion zone before scheduled block.")
                            continue
                        }
                    }
                }
                // no leader change after in exclusion zone after epoch turn over.
                if currentSlotDate.GetSlot().Cmp(jury.config.EpochTurnOverExclusionSlots) < 0 {
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
func (jury *LeaderJury) changeLeader(leaderName string) {
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
        shutDownNode(node)
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

// gets all the keys of the node map.
func getNodeNames(nodeMap map[string]Node) []string {
    var nodeNameList = make([]string, 0)
    for name, _ := range nodeMap {
        nodeNameList = append(nodeNameList, name)
    }
    return nodeNameList
}

// computes the health of all the given nodes.
func computeHealth(diffMap map[string][]*big.Int) map[string]*big.Float {
    confMap := make(map[string]*big.Float)
    for name, history := range diffMap {
        var conf *big.Float = new(big.Float)
        for i := range history {
            conf = new(big.Float).Add(conf, new(big.Float).SetInt(history[i]))
        }
        confMap[name] = conf
    }
    return confMap
}

// block height memory of all nodes for judging their health.
type blockHeightMemory struct {
    n     int
    mem   map[string][]*big.Int
    nodes []string
}

// creates a new block height memory. n specifies the number of checkpoints
// that shall be remembered. This method expects a list of nodes for which
// this memory shall be constructed is expected.
func createBlockHeightMemory(nodes []string, n int) *blockHeightMemory {
    emptyList := make([]*big.Int, n)
    for i := 0; i < n; i++ {
        emptyList[i] = new(big.Int).SetInt64(-1)
    }
    mem := make(map[string][]*big.Int)
    for i := range nodes {
        mem[nodes[i]] = emptyList
    }
    return &blockHeightMemory{n: n, mem: mem, nodes: nodes}
}

// adds the block heights for all given nodes of a new checkpoint.
func (m *blockHeightMemory) addBlockHeights(blockMap map[string]api.NodeStatistic) {
    for i := range m.nodes {
        name := m.nodes[i]
        var entry *big.Int
        stat, found := blockMap[name]
        if found {
            entry = stat.LastBlockHeight
        } else {
            entry = new(big.Int).SetInt64(-1)
        }
        m.mem[name] = append([]*big.Int{entry}, m.mem[name][:m.n]...)
    }
}

// computes the difference of the block height to the maximum reported
// block height for each of the given nodes.
func (m *blockHeightMemory) getDiff() map[string][]*big.Int {
    diffMap := make(map[string][]*big.Int)
    for n := range m.nodes {
        diffMap[m.nodes[n]] = make([]*big.Int, m.n)
    }
    for i := 0; i < m.n; i++ {
        currentMap := make(map[string]*big.Int)
        for n := range m.nodes {
            currentMap[m.nodes[n]] = m.mem[m.nodes[n]][i]
        }
        maxHeight, _ := max(currentMap)
        for name, height := range currentMap {
            diffMap[name][i] = new(big.Int).Sub(maxHeight, height)
        }
    }
    return diffMap
}

// string representation of the block height memory.
func (m *blockHeightMemory) String() string {
    var result string
    for name, num := range m.mem {
        result += fmt.Sprintf("%v=%v;", name, num)
    }
    return result + "\n"
}
