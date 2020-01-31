package monitor

import (
    log "github.com/sirupsen/logrus"
    "github.com/sobitada/go-cardano"
    jor "github.com/sobitada/go-jormungandr/api"
    "github.com/sobitada/thor/threading"
    "github.com/sobitada/thor/utils"
    "math/big"
    "sort"
    "sync"
    "time"
)

type NodeType string

const (
    Passive         NodeType = "passive"
    LeaderCandidate          = "leader-candidate"
)

type Node struct {
    // passive or leader node
    Type NodeType
    // unique name of the node
    Name string
    // api to access details about the node
    API *jor.JormungandrAPI
    // the maximal number of blocks this node
    // is allowed to lag behind.
    MaxBlockLag uint64
    // maximum time since the last block has been received.
    MaxTimeSinceLastBlock time.Duration
    // warm up time in which no shutdown shall be executed.
    WarmUpTime time.Duration
}

type NodeMonitor struct {
    nodes           []Node
    behaviour       NodeMonitorBehaviour
    actions         []Action
    ListenerManager *ListenerManager
    watchDog        *ScheduleWatchDog
    timeSettings    *cardano.TimeSettings
}

type NodeMonitorBehaviour struct {
    // interval of the monitor checking the status of nodes.
    Interval time.Duration
}

func GetNodeMonitor(nodes []Node, behaviour NodeMonitorBehaviour, actions []Action,
    watchdog *ScheduleWatchDog, settings *cardano.TimeSettings) *NodeMonitor {
    return &NodeMonitor{
        nodes:        nodes,
        behaviour:    behaviour,
        actions:      actions,
        timeSettings: settings,
        watchDog:     watchdog,
        ListenerManager: &ListenerManager{
            mutex: &sync.Mutex{},
        },
    }
}

type ListenerManager struct {
    nodeStatsListeners []chan map[string]jor.NodeStatistic
    mutex              *sync.Mutex
}

// register a listener for getting the most recent fetched node statistics for all
// monitored nodes.
func (listenerManager *ListenerManager) RegisterNodeStatisticListener(listener chan map[string]jor.NodeStatistic) {
    listenerManager.mutex.Lock()
    defer listenerManager.mutex.Unlock()
    if listenerManager.nodeStatsListeners == nil {
        listenerManager.nodeStatsListeners = []chan map[string]jor.NodeStatistic{listener}
    } else {
        listenerManager.nodeStatsListeners = append(listenerManager.nodeStatsListeners, listener)
    }
}

func getTypeAbbreviation(t NodeType) string {
    switch t {
    case Passive:
        return "PV"
    case LeaderCandidate:
        return "LC"
    default:
        return "--"
    }
}

type nodeStatisticResponse struct {
    bootstrapping bool
    nodeStats     *jor.NodeStatistic
}

func getNodeStatistics(input interface{}) threading.Response {
    node := input.(Node)
    nodeStats, bootstrapping, err := node.API.GetNodeStatistics()
    if err != nil {
        return threading.Response{
            Context: node,
            Error:   err,
        }
    } else {
        return threading.Response{
            Context: node,
            Data: &nodeStatisticResponse{
                bootstrapping: bootstrapping,
                nodeStats:     nodeStats,
            },
        }
    }
}

// a blocking call which is continuously watching
// after the Jormungandr nodes.
func (nodeMonitor *NodeMonitor) Watch() {
    log.Infof("Starting to watch nodes.")
    for ; ; {
        start := time.Now()
        // skip monitor checks before scheduled block
        if nodeMonitor.watchDog != nil {
            currentSlotDate, _ := nodeMonitor.timeSettings.GetSlotDateFor(time.Now())
            schedule, found := nodeMonitor.watchDog.GetScheduleFor(currentSlotDate.GetEpoch())
            if found {
                futureSchedule := jor.FilterLeaderLogsBefore(time.Now().Add(-2*nodeMonitor.timeSettings.SlotDuration), schedule)
                if len(futureSchedule) > 0 {
                    log.Infof("Number of leader assignments ahead: %v", len(futureSchedule))
                    log.Infof("Next leader assignments at %v", futureSchedule[0].ScheduleTime)
                    timeToNextBlock := futureSchedule[0].ScheduleTime.Sub(time.Now())
                    if timeToNextBlock < 10*nodeMonitor.timeSettings.SlotDuration {
                        time.Sleep(nodeMonitor.behaviour.Interval)
                        continue
                    }
                }
            }
        }
        // get node statistics
        blockHeightMap := make(map[string]*big.Int)
        lastBlockMap := make(map[string]jor.NodeStatistic)
        inputs := make([]interface{}, len(nodeMonitor.nodes))
        for i, node := range nodeMonitor.nodes {
            inputs[i] = node
        }
        responses := threading.Complete(inputs, getNodeStatistics)
        sort.SliceStable(responses, func(i, j int) bool {
            nodeA := responses[i].Context.(Node)
            nodeB := responses[j].Context.(Node)
            return nodeA.Name < nodeB.Name
        })
        for _, response := range responses {
            node := response.Context.(Node)
            if response.Error == nil && response.Data != nil {
                statsResponse := response.Data.(*nodeStatisticResponse)
                if !statsResponse.bootstrapping {
                    if statsResponse.nodeStats != nil {
                        lastBlockMap[node.Name] = *statsResponse.nodeStats
                        log.Infof("[%s][%s] Block Height: <%v>, Date: <%v>, Hash: <%v>, UpTime: <%v>", node.Name,
                            getTypeAbbreviation(node.Type), statsResponse.nodeStats.LastBlockHeight.String(),
                            statsResponse.nodeStats.LastBlockDate.String(),
                            statsResponse.nodeStats.LastBlockHash[:8],
                            utils.GetHumanReadableUpTime(statsResponse.nodeStats.UpTime),
                        )
                        blockHeightMap[node.Name] = statsResponse.nodeStats.LastBlockHeight
                    } else {
                        log.Errorf("[%s][%s] Node statistics cannot be fetched.", node.Name, getTypeAbbreviation(node.Type))
                    }
                } else {
                    log.Infof("[%s][%s] --- bootstrapping ---", node.Name, getTypeAbbreviation(node.Type))
                }
            } else {
                log.Errorf("[%s][%s] Node statistics cannot be fetched.", node.Name, getTypeAbbreviation(node.Type))
            }
        }
        // send block infos to leader jury
        nodeMonitor.ListenerManager.mutex.Lock()
        for i := range nodeMonitor.ListenerManager.nodeStatsListeners {
            nodeMonitor.ListenerManager.nodeStatsListeners[i] <- lastBlockMap
        }
        nodeMonitor.ListenerManager.mutex.Unlock()
        maxHeight, nodes := utils.MaxInt(blockHeightMap)
        // perform actions
        for n := range nodeMonitor.actions {
            go nodeMonitor.actions[n].execute(nodeMonitor.nodes, ActionContext{
                TimeSettings:         nodeMonitor.timeSettings,
                BlockHeightMap:       blockHeightMap,
                MaximumBlockHeight:   maxHeight,
                UpToDateNodes:        nodes,
                LastNodeStatisticMap: lastBlockMap,
            })
        }
        diff := start.Add(nodeMonitor.behaviour.Interval).Sub(time.Now())
        if diff > 0 {
            time.Sleep(diff)
        }
    }
}
