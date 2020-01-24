package monitor

import (
    log "github.com/sirupsen/logrus"
    "github.com/sobitada/go-cardano"
    jor "github.com/sobitada/go-jormungandr/api"
    "github.com/sobitada/thor/pooltool"
    "github.com/sobitada/thor/utils"
    "math/big"
    "sync"
    "time"
)

type NodeType byte

const (
    Passive NodeType = iota
    Leader
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
    poolTool        *pooltool.PoolTool
    watchDog        *ScheduleWatchDog
    timeSettings    *cardano.TimeSettings
}

type NodeMonitorBehaviour struct {
    // interval of the monitor checking the status of
    // nodes.
    IntervalInMs uint32
}

func GetNodeMonitor(nodes []Node, behaviour NodeMonitorBehaviour, actions []Action, poolTool *pooltool.PoolTool,
    watchdog *ScheduleWatchDog, settings *cardano.TimeSettings) *NodeMonitor {
    return &NodeMonitor{
        nodes:        nodes,
        behaviour:    behaviour,
        actions:      actions,
        timeSettings: settings,
        watchDog:     watchdog,
        poolTool:     poolTool,
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

// a blocking call which is continuously watching
// after the Jormungandr nodes.
func (nodeMonitor *NodeMonitor) Watch() {
    log.Infof("Starting to watch nodes.")
    for ; ; {
        blockHeightMap := make(map[string]*big.Int)
        lastBlockMap := make(map[string]jor.NodeStatistic)
        names := make([]string, len(nodeMonitor.nodes))
        for i := range nodeMonitor.nodes {
            // skip monitor checks before scheduled block
            if nodeMonitor.watchDog != nil {
                currentSlotDate, _ := nodeMonitor.timeSettings.GetSlotDateFor(time.Now())
                schedule, found := nodeMonitor.watchDog.GetScheduleFor(currentSlotDate.GetEpoch())
                if found {
                    futureSchedule := jor.FilterLeaderLogsBefore(time.Now().Add(-2*nodeMonitor.timeSettings.SlotDuration), schedule)
                    if len(futureSchedule) > 0 {
                        timeToNextBlock := futureSchedule[0].ScheduleTime.Sub(time.Now())
                        if timeToNextBlock < 10*nodeMonitor.timeSettings.SlotDuration {
                            time.Sleep(time.Duration(nodeMonitor.behaviour.IntervalInMs) * time.Millisecond)
                            continue
                        }
                    }
                }

            }
            // monitor checks
            node := nodeMonitor.nodes[i]
            names[i] = node.Name
            nodeStats, bootstrapping, err := nodeMonitor.nodes[i].API.GetNodeStatistics()
            if err == nil {
                if !bootstrapping {
                    if nodeStats != nil {
                        lastBlockMap[node.Name] = *nodeStats
                        log.Infof("[%s] Block Height: <%v>, Date: <%v>, Hash: <%v>, UpTime: <%v>", node.Name, nodeStats.LastBlockHeight.String(),
                            nodeStats.LastBlockDate.String(),
                            nodeStats.LastBlockHash[:8],
                            utils.GetHumanReadableUpTime(nodeStats.UpTime),
                        )
                        blockHeightMap[node.Name] = nodeStats.LastBlockHeight
                    } else {
                        log.Errorf("[%s] Node details cannot be fetched.", node.Name)
                    }
                } else {
                    log.Infof("[%s] --- bootstrapping ---", node.Name)
                }
            } else {
                log.Errorf("[%s] Node details cannot be fetched.", node.Name)
            }
        }
        // send block infos to leader jury
        nodeMonitor.ListenerManager.mutex.Lock()
        for i := range nodeMonitor.ListenerManager.nodeStatsListeners {
            nodeMonitor.ListenerManager.nodeStatsListeners[i] <- lastBlockMap
        }
        nodeMonitor.ListenerManager.mutex.Unlock()
        maxHeight, nodes := utils.MaxInt(blockHeightMap)
        // update pool tool if configured
        if nodeMonitor.poolTool != nil {
            nodeMonitor.poolTool.PushLatestTip(maxHeight)
        }
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
        time.Sleep(time.Duration(nodeMonitor.behaviour.IntervalInMs) * time.Millisecond)
    }
}
