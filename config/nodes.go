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
    // type of the node, "passive" or "leader-candidate"
    Type monitor.NodeType `yaml:"type"`
    // unique name for a node.
    Name string `yaml:"name"`
    // URL to the API of a node
    APIUrl string `yaml:"api"`
    // maximum number of blocks a node can lag behind.
    MaxBlockLag uint64 `yaml:"maxBlockLag"`
    // maximum time in ms since the last block has been received.
    MaxTimeSinceLastBlockInMs int64 `yaml:"maxTimeSinceLastBlock"`
    // warm up time in which no shutdown shall be executed.
    WarmUpTime int64 `yaml:"warmUpTime"`
    // timeout in milliseconds for API calls to this node.
    Timeout uint32 `yaml:"apiTimeout"`
}

// extracts the node details from the configuration file.
func GetNodesFromConfig(config General) ([]monitor.Node, error) {
    nodeList := make([]monitor.Node, 0)
    nodeNameMap := make(map[string]bool)
    for i := range config.Peers {
        peerConfig := config.Peers[i]
        _, found := nodeNameMap[peerConfig.Name]
        if found {
            return nil, ConfigurationError{
                Path:   "peers/name",
                Reason: "The name of a peer must be unique.",
            }
        } else {
            nodeNameMap[peerConfig.Name] = true
        }
        // API timeout
        var apiTimeout time.Duration
        if peerConfig.Timeout == 0 {
            apiTimeout = 3 * time.Second
        } else {
            apiTimeout = time.Duration(peerConfig.Timeout) * time.Millisecond
        }
        api, err := jor.GetAPIFromHost(peerConfig.APIUrl, apiTimeout)
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
            t := peerConfig.Type
            if t == "" {
                t = monitor.Passive
            }
            nodeList = append(nodeList, monitor.Node{
                Type:                  t,
                Name:                  peerConfig.Name,
                API:                   api,
                MaxBlockLag:           peerConfig.MaxBlockLag,
                MaxTimeSinceLastBlock: maxTimeSinceLastBlock,
                WarmUpTime:            time.Duration(peerConfig.WarmUpTime) * time.Millisecond,
            })
        } else {
            log.Warn("[%s] Could not build an API for this peer from the specified configuration. %s", err.Error())
        }
    }
    return nodeList, nil
}
