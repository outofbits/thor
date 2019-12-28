package config

// general config for this application.
type General struct {
    Logging  Logging  `yaml:"logging"`
    Peers    []Node   `yaml:"peers"`
    Monitor  Monitor  `yaml:"monitor"`
    PoolTool *PoolTool `yaml:"pooltool"`
}
