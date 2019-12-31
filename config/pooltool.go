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
}

func ParsePostLastTipToPoolToolAction(conf General) (*monitor.PoolToolActionConfig, error) {
    if conf.PoolTool != nil {
        poolToolConf := *conf.PoolTool
        if poolToolConf.UserID != "" && poolToolConf.PoolID != "" {
            if conf.Blockchain != nil && conf.Blockchain.GenesisBlockHash != "" {
                return &monitor.PoolToolActionConfig{
                    PoolID:      poolToolConf.PoolID,
                    UserID:      poolToolConf.UserID,
                    GenesisHash: conf.Blockchain.GenesisBlockHash,
                }, nil
            } else {
                return nil, ConfigurationError{Path: "blockchain/genesisBlockHash", Reason: "The hash of the genesis block must be specified for Pool Tool actions."}
            }
        } else {
            return nil, ConfigurationError{Path: "pooltool", Reason: "Personal pool ID, pool tool user ID  must be specified."}
        }
    }
    return nil, nil
}
