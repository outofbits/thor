package config

import (
    log "github.com/sirupsen/logrus"
    jor "github.com/sobitada/go-jormungandr/api"
    "github.com/sobitada/thor/monitor"
    "time"
)

// configuration struct for the nodes that shall
// be watched.
type Node struct {
    // unique name for a node.
    Name string `yaml:"name"`
    // URL to the API of a node
    APIUrl string `yaml:"api"`
    // maximum number of blocks a node can lag behind.
    MaxBlockLag uint64 `yaml:"maxBlockLag"`
    // maximum time in ms since the last block has been received.
    MaxTimeSinceLastBlockInMs int64 `yaml:"maxTimeSinceLastBlock"`
}

// extracts the node details from the configuration file.
func GetNodesFromConfig(config General) []monitor.Node {
    nodeList := make([]monitor.Node, 0)
    for i := range config.Peers {
        peerConfig := config.Peers[i]
        api, err := jor.GetAPIFromHost(peerConfig.APIUrl)
        if err == nil {
            var maxTimeSinceLastBlock time.Duration
            // maximum block lag.
            if peerConfig.MaxBlockLag == 0 {
                log.Warnf("Node '%v' has not set any maximum lag for the block height.", peerConfig.Name)
            }
            // maximum time since last block has been received setting.
            if peerConfig.MaxTimeSinceLastBlockInMs > 0 {
                maxTimeSinceLastBlock = time.Duration(peerConfig.MaxTimeSinceLastBlockInMs) * time.Millisecond
            } else {
                log.Warnf("Node '%v' has not set any maximum time since new block has been received.", peerConfig.Name)
            }
            nodeList = append(nodeList, monitor.Node{
                Type:                  monitor.Passive,
                Name:                  peerConfig.Name,
                API:                   api,
                MaxBlockLag:           peerConfig.MaxBlockLag,
                MaxTimeSinceLastBlock: maxTimeSinceLastBlock,
            })
        } else {
            log.Warn("[%s] Could not build an API for this peer from the specified configuration. %s", err.Error())
        }
    }
    return nodeList
}
