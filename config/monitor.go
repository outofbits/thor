package config

import (
    "github.com/sobitada/thor/monitor"
    "time"
)

// configuration struct for the monitor settings.
type Monitor struct {
    // interval in which the status of nodes shall be checked.
    // it must be specified in milliseconds.
    IntervalInMs uint32 `yaml:"interval"`
    // needed for enabling the leader election jury.
    LeaderConfig *LeaderConfig `yaml:"leaderJury"`
}

// gets the behaviour of the monitor specified in the given configuration.
func GetNodeMonitorBehaviour(config General) monitor.NodeMonitorBehaviour {
    var interval time.Duration
    if config.Monitor.IntervalInMs == 0 {
        interval = 60 * time.Second
    } else {
        interval = time.Duration(config.Monitor.IntervalInMs) * time.Millisecond
    }
    return monitor.NodeMonitorBehaviour{Interval: interval}
}
