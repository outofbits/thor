package config

import (
    "github.com/sobitada/go-cardano"
    "github.com/sobitada/go-jormungandr/api"
    "github.com/sobitada/thor/leader"
    "github.com/sobitada/thor/monitor"
    "io/ioutil"
    "math/big"
    "time"
)

type LeaderConfig struct {
    CertPath                    string `yaml:"cert"`
    Window                      int    `yaml:"window"`
    ExclusionZoneInS            uint32 `yaml:"exclusionZone"`
    PreTurnOverExclusionZoneInS uint32 `yaml:"preTurnoverExclusionZone"`
}

// gets the leader jury for the given configuration. It expects also the nodes
// for which the leader jury shall be activated.
func GetLeaderJury(nodes []monitor.Node, mon *monitor.NodeMonitor, watchDog *monitor.ScheduleWatchDog,
    timeSettings *cardano.TimeSettings, config General) (*leader.Jury, error) {
    leaderConfig := config.Monitor.LeaderConfig
    if leaderConfig != nil {
        if timeSettings != nil {
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
                        if leaderConfig.ExclusionZoneInS == 0 {
                            exclusionZone = 30 * time.Second
                        } else {
                            exclusionZone = time.Duration(int64(leaderConfig.ExclusionZoneInS)) * time.Second
                        }
                        // pre epoch turn over exclusion zone for leader change.
                        var preTurnOverExclusionSlots *big.Int
                        var preTurnOverExclusionInS uint32 = 60
                        if leaderConfig.PreTurnOverExclusionZoneInS > 0 {
                            preTurnOverExclusionInS = leaderConfig.PreTurnOverExclusionZoneInS
                        }
                        preTurnOverExclusionSlots = new(big.Int).Div(new(big.Int).SetInt64(int64(time.Duration(int64(preTurnOverExclusionInS))*time.Second)),
                            new(big.Int).SetInt64(int64(timeSettings.SlotDuration)))
                        return leader.GetLeaderJuryFor(nodes, mon, watchDog, leaderCert, leader.JurySettings{
                            Window:                         window,
                            ExclusionZone:                  exclusionZone,
                            PreEpochTurnOverExclusionSlots: preTurnOverExclusionSlots,
                            TimeSettings:                   timeSettings,
                        })
                    } else {
                        return nil, ConfigurationError{Path: "monitor/leader_jury/cert", Reason: err.Error()}
                    }
                } else {
                    return nil, ConfigurationError{Path: "monitor/leader_jury/cert", Reason: err.Error()}
                }
            } else {
                return nil, ConfigurationError{Path: "monitor/leader_jury/cert", Reason: "The certification path must be specified."}
            }
        } else {
            return nil, ConfigurationError{Path: "monitor/leader_jury", Reason: "You must specify blockchain settings to use leader jury."}
        }
    }
    return nil, nil
}
