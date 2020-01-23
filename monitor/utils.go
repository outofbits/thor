package monitor

import "time"

// gets all the node names of a node map.
func GetNodeNames(nodeMap map[string]Node) []string {
    var nodeNameList = make([]string, 0)
    for name, _ := range nodeMap {
        nodeNameList = append(nodeNameList, name)
    }
    return nodeNameList
}

// shuts down the given node.
func ShutDownNode(node Node) {
    _ = node.API.Shutdown()
    time.Sleep(time.Duration(200) * time.Millisecond)
    _ = node.API.Shutdown()
}