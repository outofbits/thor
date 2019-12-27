package config

import (
	log "github.com/sirupsen/logrus"
	jor "github.com/sobitada/go-jormungandr/wrapper"
	"github.com/sobitada/thor/monitor"
)

type PeerConfig struct {
	Name        string `yaml:"name"`
	APIUrl      string `yaml:"api"`
	MaxBlockLag uint32 `yaml:"maxBlockLag"`
}

// extracts the node details from the configuration file.
func GetNodesFromConfig(config Config) []monitor.Node {
	nodeList := make([]monitor.Node, 0)
	// peer nodes
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
