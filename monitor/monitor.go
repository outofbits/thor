package monitor

import (
    log "github.com/sirupsen/logrus"
    "github.com/sobitada/go-cardano"
    jor "github.com/sobitada/go-jormungandr/api"
    "github.com/sobitada/thor/pooltool"
    "math/big"
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

type NodeMonitor interface {
    // a blocking call which is continuously watching
    // after the Jormungandr nodes.
    Watch()
}

type NodeMonitorBehaviour struct {
    // interval of the monitor checking the status of
    // nodes.
    IntervalInMs uint32
}

type nodeMonitorImpl struct {
    Nodes        []Node
    Behaviour    NodeMonitorBehaviour
    Actions      []Action
    TimeSettings *cardano.TimeSettings
    PoolTool     *pooltool.PoolTool
    LeaderJury   *LeaderJury
}

func GetNodeMonitor(nodes []Node, behaviour NodeMonitorBehaviour, actions []Action, settings *cardano.TimeSettings, poolTool *pooltool.PoolTool, jury *LeaderJury) NodeMonitor {
    return nodeMonitorImpl{Nodes: nodes, Behaviour: behaviour, Actions: actions, TimeSettings: settings, PoolTool: poolTool, LeaderJury: jury}
}

func (nodeMonitor nodeMonitorImpl) RegisterAction(action Action) {
    nodeMonitor.Actions = append(nodeMonitor.Actions, action)
}

func (nodeMonitor nodeMonitorImpl) Watch() {
    log.Infof("Starting to watch nodes.")
    for ; ; {
        blockHeightMap := make(map[string]*big.Int)
        lastBlockMap := make(map[string]jor.NodeStatistic)
        names := make([]string, len(nodeMonitor.Nodes))
        for i := range nodeMonitor.Nodes {
            node := nodeMonitor.Nodes[i]
            names[i] = node.Name
            nodeStats, bootstrapping, err := nodeMonitor.Nodes[i].API.GetNodeStatistics()
            if err == nil {
                if !bootstrapping {
                    if nodeStats != nil {
                        lastBlockMap[node.Name] = *nodeStats
                        log.Infof("[%s] Block Height: <%v>, Date: <%v>, Hash: <%v>, UpTime: <%v>", node.Name, nodeStats.LastBlockHeight.String(),
                            nodeStats.LastBlockDate.String(),
                            nodeStats.LastBlockHash[:8],
                            getHumanReadableUpTime(nodeStats.UpTime),
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
        if nodeMonitor.LeaderJury != nil {
            nodeMonitor.LeaderJury.BlockStatsChannel <- lastBlockMap
        }
        maxHeight, nodes := max(blockHeightMap)
        // update pool tool if configured
        if nodeMonitor.PoolTool != nil {
            nodeMonitor.PoolTool.PushLatestTip(maxHeight)
        }
        // perform actions
        for n := range nodeMonitor.Actions {
            go nodeMonitor.Actions[n].execute(nodeMonitor.Nodes, ActionContext{
                TimeSettings:         nodeMonitor.TimeSettings,
                BlockHeightMap:       blockHeightMap,
                MaximumBlockHeight:   maxHeight,
                UpToDateNodes:        nodes,
                LastNodeStatisticMap: lastBlockMap,
            })
        }
        time.Sleep(time.Duration(nodeMonitor.Behaviour.IntervalInMs) * time.Millisecond)
    }
}
