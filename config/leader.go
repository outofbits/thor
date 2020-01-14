package config

import (
    "github.com/sobitada/go-jormungandr/api"
    "github.com/sobitada/thor/monitor"
    "io/ioutil"
    "time"
)

type LeaderConfig struct {
    CertPath      string `yaml:"cert"`
    Window        int    `yaml:"window"`
    ExclusionZone uint32 `yaml:"exclusion_zone"`
}

// gets the leader jury for the given configuration. It expects also the nodes
// for which the leader jury shall be activated.
func GetLeaderJury(nodes []monitor.Node, config General) (*monitor.LeaderJury, error) {
    leaderConfig := config.Monitor.LeaderConfig
    if leaderConfig != nil {
        if leaderConfig.CertPath != "" {
            certData, err := ioutil.ReadFile(leaderConfig.CertPath)
            if err == nil {
                leaderCert, err := api.ReadLeaderCertificate(certData)
                if err == nil {
                    // checkpoints window
                    var window = leaderConfig.Window
                    if window <= 0 {
                        window = 5
                    }
                    // exclusion zone for leader change.
                    var exclusionZone time.Duration
                    if leaderConfig.ExclusionZone == 0 {
                        exclusionZone = 30 * time.Second
                    } else {
                        exclusionZone = time.Duration(int64(leaderConfig.ExclusionZone)) * time.Second
                    }
                    return monitor.GetLeaderJuryFor(nodes, leaderCert, monitor.LeaderJuryConfig{
                        Window:        window,
                        ExclusionZone: exclusionZone,
                    }), nil
                } else {
                    return nil, ConfigurationError{Path: "monitor/leader_jury/cert", Reason: err.Error()}
                }
            } else {
                return nil, ConfigurationError{Path: "monitor/leader_jury/cert", Reason: err.Error()}
            }
        } else {
            return nil, ConfigurationError{Path: "monitor/leader_jury/cert", Reason: "The certification path must be specified."}
        }
    }
    return nil, nil
}
