package config

import "github.com/sobitada/thor/monitor"

// configuration struct for the monitor settings.
type Monitor struct {
    // interval in which the status of nodes shall be checked.
    // it must be specified in milliseconds.
    IntervalInMs uint32 `yaml:"interval"`
}

// gets the behaviour of the monitor specified in the given configuration.
func GetNodeMonitorBehaviour(config General) monitor.NodeMonitorBehaviour {
    return monitor.NodeMonitorBehaviour{IntervalInMs: config.Monitor.IntervalInMs}
}
