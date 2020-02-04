package config

import (
    "github.com/sobitada/thor/monitor"
    "github.com/sobitada/thor/prometheus"
)

type Prometheus struct {
    Hostname string `yaml:"hostname"`
    Port     string `yaml:"port"`
}

func ParsePrometheusConfig(mon *monitor.NodeMonitor, conf General) (*prometheus.Client, error) {
    if conf.Prometheus != nil {
        prometheusConf := *conf.Prometheus
        if prometheusConf.Hostname != "" && prometheusConf.Port != "" {
            return prometheus.GetClient(prometheusConf.Hostname, prometheusConf.Port, mon), nil
        } else {
            return nil, ConfigurationError{Path: "prometheus", Reason: "Hostname and port must be specified for Prometheus."}
        }
    }
    return nil, nil
}
