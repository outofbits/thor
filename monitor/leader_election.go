package monitor

import (
    "fmt"
    log "github.com/sirupsen/logrus"
    "github.com/sobitada/go-jormungandr/api"
    "math/big"
    "strings"
    "time"
)

type LeaderJury struct {
    leader            *currentLeader
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
}

// gets the leader jury judging the given nodes. it expects the certificate of the
// leader that shall be managed and configuration details.
func GetLeaderJuryFor(nodes []Node, certificate api.LeaderCertificate, config LeaderJuryConfig) *LeaderJury {
    c := make(chan map[string]api.NodeStatistic)
    nodeMap := make(map[string]Node)
    for i := range nodes {
        nodeMap[nodes[i].Name] = nodes[i]
    }
    return &LeaderJury{
        nodes:             nodeMap,
        BlockStatsChannel: c,
        Cert:              certificate,
        config:            config,
    }
}

// scans for the current leader among all the nodes,
// it expects only one leader node. multi leaders are
// not supported at the moment.
func (jury *LeaderJury) scanForLeader() *currentLeader {
    for name, node := range jury.nodes {
        leaders, err := node.API.GetRegisteredLeaders()
        if err == nil {
            if len(leaders) > 0 {
                return &currentLeader{name: name, leaderID: leaders[0]}
            }
        }
    }
    return nil
}

//
func getSchedule(node Node, epoch *big.Int, scheduleMap *map[string][]api.LeaderAssignment) []api.LeaderAssignment {
    _, found := (*scheduleMap)[epoch.String()]
    if !found {
        schedule, err := node.API.GetLeadersSchedule()
        if err == nil {
            (*scheduleMap)[epoch.String()] = api.SortLeaderLogsByScheduleTime(schedule)
        } else {
            (*scheduleMap)[epoch.String()] = []api.LeaderAssignment{}
        }
    }
    return (*scheduleMap)[epoch.String()]
}

// starts the leader jury and let it continuously run. it reads all the checkpoints
// that have been passed from the monitor to this leader jury.
func (jury *LeaderJury) Judge() {
    leader := jury.scanForLeader()
    if leader != nil {
        log.Infof("[LEADER JURY] Node %v is elected and has ID=%v", leader.name, leader.leaderID)
        jury.leader = leader
    }
    nodeNames := getNodeNames(jury.nodes)
    mem := createBlockHeightMemory(nodeNames, jury.config.Window)
    scheduleMap := make(map[string][]api.LeaderAssignment)
    for ; ; {
        latestBlockStats := <-jury.BlockStatsChannel
        // check leader schedule
        var schedule []api.LeaderAssignment
        for key, value := range latestBlockStats {
            schedule = getSchedule(jury.nodes[key], value.LastBlockDate.GetEpoch(), &scheduleMap)
            break
        }
        //
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
                            continue
                        }
                    }
                }
                // no leader change after epoch turn over.
                // TODO: safety check for epoch turn over.
                // change leader. //TODO: make it random, do not sort!
                jury.changeLeader(maxConfNodes[0])
            }
        }
        if jury.leader != nil {
            log.Infof("[LEADER JURY] Current Leader is %v.", jury.leader.name)
        }
    }
}

// changes the leader to the given name.
func (jury *LeaderJury) changeLeader(leaderName string) {
    newLeader := jury.nodes[leaderName]
    leaderID, err := newLeader.API.PostLeader(jury.Cert)
    if err == nil {
        if jury.leader != nil {
            found, err := jury.nodes[jury.leader.name].API.RemoveRegisteredLeader(jury.leader.leaderID)
            if err != nil {
                log.Warnf("[LEADER JURY] The leader node %v could not be demoted. %v", jury.leader.name, err.Error())
            } else if !found {
                log.Warnf("[LEADER JURY] The leader node %v was not in leader mode.", jury.leader.name)
            }
        }
        jury.leader = &currentLeader{name: newLeader.Name, leaderID: leaderID}
        log.Infof("[LEADER JURY] Node %v is elected and has ID=%v", newLeader.Name, leaderID)
    } else {
        log.Errorf("[LEADER JURY] Could not change to leader %v. %v", newLeader.Name, err.Error())
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
