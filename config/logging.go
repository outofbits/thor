package config

// configuration struct for the logging settings.
type Logging struct {
    // logging level that shall be used, levels can be panic,
    // fatal, error, warn, info, debug or trace.
    Level string `yaml:"level"`
    // configuration to send the logs to
    // graylog.
    GrayLog *GrayLog `yaml:"graylog"`
}

type GrayLog struct {
    Host string `yaml:"host"`
    Port string `yaml:"port"`
}
