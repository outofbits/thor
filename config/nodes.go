package config

import (
    log "github.com/sirupsen/logrus"
    jor "github.com/sobitada/go-jormungandr/wrapper"
    "github.com/sobitada/thor/monitor"
)

// configuration struct for the nodes that shall
// be watched.
type Node struct {
    // unique name for a node.
    Name string `yaml:"name"`
    // URL to the API of a node
    APIUrl string `yaml:"api"`
    // maximum number of blocks a node can lag behind.
    MaxBlockLag uint32 `yaml:"maxBlockLag"`
}

// extracts the node details from the configuration file.
func GetNodesFromConfig(config General) []monitor.Node {
    nodeList := make([]monitor.Node, 0)
    for i := range config.Peers {
        peerConfig := config.Peers[i]
        api, err := jor.GetAPIFromHost(peerConfig.APIUrl)
        if err == nil {
            nodeList = append(nodeList, monitor.Node{Type: monitor.Passive, Name: peerConfig.Name, API: api})
        } else {
            log.Warn("[%s] Could not build an API for this peer from the specified configuration. %s", err.Error())
        }
    }
    return nodeList
}
