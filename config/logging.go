package config

// configuration struct for the logging settings.
type Logging struct {
    // logging level that shall be used, levels can be panic,
    // fatal, error, warn, info, debug or trace.
    Level string `yaml:"level"`
}
