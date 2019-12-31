package config

import (
    "github.com/sobitada/go-jormungandr/cardano"
    "time"
)

type BlockchainSettings struct {
    GenesisBlockHash     string    `yaml:"genesisBlockHash"`
    GenesisBlockDateTime time.Time `yaml:"genesisBlockTime"`
    SlotsPerEpoch        uint64    `yaml:"slotsPerEpoch"`
    SlotDurationInMs     uint64    `yaml:"slotDuration"`
}

func GetTimeSettings(conf BlockchainSettings) (*cardano.TimeSettings, error) {
    if conf.SlotsPerEpoch > 0 && conf.SlotDurationInMs > 0 {
        return &cardano.TimeSettings{
            GenesisBlockDateTime: conf.GenesisBlockDateTime,
            SlotsPerEpoch:        conf.SlotsPerEpoch,
            SlotDuration:         time.Millisecond * time.Duration(conf.SlotDurationInMs),
        }, nil
    } else {
        return nil, ConfigurationError{
            Path:   "blockchain",
            Reason: "Time Settings cannot be established, because slots per epoch or/and duration was not specified.",
        }
    }
}
