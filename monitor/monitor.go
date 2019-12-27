package monitor

import (
	log "github.com/sirupsen/logrus"
	jor "github.com/sobitada/go-jormungandr/wrapper"
	"strconv"
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
	API jor.JormungandrAPI
	// the maximal number of blocks this node
	// is allowed to lag behind.
	MaxBlockLag uint32
}

type NodeMonitor interface {
	// a blocking call which is continuously watching
	// after the Jormungandr nodes.
	Watch()
	// registers a new action that is executed at the end
	// of each interval.
	RegisterAction(action Action)
}

type NodeMonitorBehaviour struct {
	// interval of the monitor checking the status of
	// nodes.
	IntervalInMs uint32
}

type nodeMonitorImpl struct {
	Nodes     []Node
	Behaviour NodeMonitorBehaviour
	Actions   []Action
}

func GetNodeMonitor(nodes []Node, behaviour NodeMonitorBehaviour) NodeMonitor {
	return nodeMonitorImpl{Nodes: nodes, Behaviour: behaviour}
}

func (nodeMonitor nodeMonitorImpl) RegisterAction(action Action) {
	nodeMonitor.Actions = append(nodeMonitor.Actions, action)
}

func (nodeMonitor nodeMonitorImpl) Watch() {
	log.Infof("Starting to monitor peers", )
	for ; ; {
		blockHeightMap := make(map[string]uint32)
		for i := range nodeMonitor.Nodes {
			node := nodeMonitor.Nodes[i]
			nodeStats, err := nodeMonitor.Nodes[i].API.GetNodeStatistics()
			if err == nil && nodeStats != nil {
				if nodeStats.LastBlockHeight != "" {
					log.Infof("[%s] Block Height: <%v>, Date: <%v>, Hash: <%v>", node.Name, nodeStats.LastBlockHeight, nodeStats.LastBlockDate, nodeStats.LastBlockHash[:8])
					bH, err := strconv.Atoi(nodeStats.LastBlockHeight)
					if err == nil {
						blockHeightMap[node.Name] = uint32(bH)
					}
				} else {
					log.Infof("[%s] ---", node.Name)
				}
			} else {
				log.Errorf("[%s] Node details cannot be fetched.", node.Name)
			}
		}
		for n := range nodeMonitor.Actions {
			go nodeMonitor.Actions[n].execute(nodeMonitor.Nodes, ActionContext{BlockHeightMap: blockHeightMap})
		}
		time.Sleep(time.Duration(nodeMonitor.Behaviour.IntervalInMs) * time.Millisecond)
	}
}
