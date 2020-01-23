package config

import (
    "github.com/sobitada/go-cardano"
    "github.com/sobitada/go-jormungandr/api"
    "github.com/sobitada/thor/monitor"
    "io/ioutil"
    "math/big"
    "time"
)

type LeaderConfig struct {
    CertPath                 string `yaml:"cert"`
    Window                   int    `yaml:"window"`
    ExclusionZoneInS         uint32 `yaml:"exclusion_zone"`
    TurnOverExclusionZoneInS uint32 `yaml:"turnover_exclusion_zone"`
}

// gets the leader jury for the given configuration. It expects also the nodes
// for which the leader jury shall be activated.
func GetLeaderJury(nodes []monitor.Node, timeSettings *cardano.TimeSettings, config General) (*monitor.Jury, error) {
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
                        // epoch turn over exclusion zone for leader change.
                        var turnOverExclusionSlots *big.Int
                        var turnOverExclusionInS uint32 = 600
                        if leaderConfig.TurnOverExclusionZoneInS > 0 {
                            turnOverExclusionInS = leaderConfig.TurnOverExclusionZoneInS
                        }
                        turnOverExclusionSlots = new(big.Int).Div(new(big.Int).SetInt64(int64(time.Duration(int64(turnOverExclusionInS))*time.Second)),
                            new(big.Int).SetInt64(int64(timeSettings.SlotDuration)))
                        return monitor.GetLeaderJuryFor(nodes, leaderCert, monitor.JurySettings{
                            Window:                      window,
                            ExclusionZone:               exclusionZone,
                            EpochTurnOverExclusionSlots: turnOverExclusionSlots,
                            TimeSettings:                timeSettings,
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
        } else {
            return nil, ConfigurationError{Path: "monitor/leader_jury", Reason: "You must specify blockchain settings to use leader jury."}
        }
    }
    return nil, nil
}
