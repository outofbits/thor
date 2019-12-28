package config

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
