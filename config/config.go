package config

import "fmt"

// general config for this application.
type General struct {
    Logging      Logging             `yaml:"logging"`
    Blockchain   *BlockchainSettings `yaml:"blockchain"`
    Peers        []Node              `yaml:"peers"`
    Monitor      Monitor             `yaml:"monitor"`
    PoolTool     *PoolTool           `yaml:"pooltool"`
    Email        *Email              `yaml:"email"`
}

type ConfigurationError struct {
    // path to the configuration at which an error is located.
    Path string
    // Reason of the error
    Reason string
}

func (error ConfigurationError) Error() string {
    return fmt.Sprintf("Configuration error at '%v'. %v", error.Path, error.Reason)
}
