package config

import "github.com/sobitada/thor/monitor"

type Config struct {
    Logging  LoggingConfig `yaml:"logging"`
    Peers    []PeerConfig  `yaml:"peers"`
    Monitor  MonitorConfig `yaml:"monitor"`
    PoolTool PoolTool      `yaml:"pooltool"`
}

type LoggingConfig struct {
    Level string `yaml:"level"`
}

type MonitorConfig struct {
    IntervalInMs uint32 `yaml:"interval"`
}

func GetNodeMonitorBehaviour(config Config) monitor.NodeMonitorBehaviour {
    return monitor.NodeMonitorBehaviour{IntervalInMs: config.Monitor.IntervalInMs}
}
