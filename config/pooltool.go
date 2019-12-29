package config

import (
    "github.com/sobitada/thor/monitor"
)

// configuration struct for PoolTool
type PoolTool struct {
    // user ID in PoolTool
    UserID string `yaml:"userID"`
    // the ID of the pool for which information
    // shall be requested/pushed from PoolTool.
    PoolID string `yaml:"poolID"`
    // the genesis hash of the blockchain for which
    // PoolTool shall be used.
    GenesisHash string `yaml:"genesisHash"`
}

func ParsePostLastTipToPoolToolAction(conf General) (*monitor.PoolToolActionConfig, error) {
    if conf.PoolTool != nil {
        poolToolConf := *conf.PoolTool
        if poolToolConf.UserID != "" && poolToolConf.GenesisHash != "" && poolToolConf.PoolID != "" {
            return &monitor.PoolToolActionConfig{
                PoolID:      poolToolConf.PoolID,
                UserID:      poolToolConf.UserID,
                GenesisHash: poolToolConf.GenesisHash,
            }, nil
        } else {
            return nil, ConfigurationError{Path: "pooltool", Reason: "Personal pool ID, pool tool user ID as well as genesis hash of the blockchain must be specified."}
        }
    }
    return nil, nil
}
