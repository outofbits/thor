package prometheus

import (
    "fmt"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    log "github.com/sirupsen/logrus"
    jor "github.com/sobitada/go-jormungandr/api"
    "github.com/sobitada/thor/monitor"
    "math/big"
    "net/http"
)

type Client struct {
    host                 string
    port                 string
    nodeStatisticChannel chan map[string]jor.NodeStatistic
}

func GetClient(host string, port string, mon *monitor.NodeMonitor) *Client {
    nodeStatisticChannel := make(chan map[string]jor.NodeStatistic)
    mon.ListenerManager.RegisterNodeStatisticListener(nodeStatisticChannel)
    return &Client{host: host, port: port, nodeStatisticChannel: nodeStatisticChannel}
}

var (
    lastBlockHeight = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "thor_jormungandr_last_block_height",
            Help: "The latest block height reported by this Jormungandr node.",
        }, []string{
            "name",
            "version",
        })
    transactionReceivedCount = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "thor_jormungandr_tx_received_count",
            Help: "The number of transaction received by this Jormungandr node.",
        }, []string{
            "name",
            "version",
        })
    peerAvailableCount = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "thor_jormungandr_peer_available_count",
            Help: "The number of peers available to this Jormungandr node.",
        }, []string{
            "name",
            "version",
        })
    peerQuarantinedCount = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "thor_jormungandr_peer_quarantined_count",
            Help: "The number of peers quarantined to this Jormungandr node.",
        }, []string{
            "name",
            "version",
        })
    peerUnreachableCount = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "thor_jormungandr_peer_unreachable_count",
            Help: "The number of peers unreachable to this Jormungandr node.",
        }, []string{
            "name",
            "version",
        })
    upTime = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "thor_jormungandr_uptime",
            Help: "The uptime reported by this jormungandr node.",
        }, []string{
            "name",
            "version",
        })
)

func (client *Client) update() {
    for ; ; {
        nodeStatisticMap := <-client.nodeStatisticChannel
        for name, value := range nodeStatisticMap {
            if value.LastBlockHeight != nil {
                height, _ := new(big.Float).SetInt(value.LastBlockHeight).Float64()
                lastBlockHeight.WithLabelValues(name, value.JormungandrVersion).Set(height)
            }
            if value.PeerAvailableCount != nil {
                peerAvailableCount.WithLabelValues(name, value.JormungandrVersion).Set(float64(*value.PeerAvailableCount))
            }
            if value.PeerQuarantinedCount != nil {
                peerQuarantinedCount.WithLabelValues(name, value.JormungandrVersion).Set(float64(*value.PeerQuarantinedCount))
            }
            if value.PeerUnreachableCnt != nil {
                peerUnreachableCount.WithLabelValues(name, value.JormungandrVersion).Set(float64(*value.PeerUnreachableCnt))
            }
            upTime.WithLabelValues(name, value.JormungandrVersion).Set(value.UpTime.Seconds())
        }
    }
}

func (client *Client) Run() {
    prometheus.MustRegister(lastBlockHeight)
    prometheus.MustRegister(transactionReceivedCount)
    prometheus.MustRegister(peerAvailableCount)
    prometheus.MustRegister(peerQuarantinedCount)
    prometheus.MustRegister(peerUnreachableCount)
    prometheus.MustRegister(upTime)
    http.Handle("/metrics", promhttp.Handler())
    go client.update()
    err := http.ListenAndServe(fmt.Sprintf("%v:%v", client.host, client.port), nil)
    if err != nil {
        log.Errorf("Prometheus client could not be started. %v", err.Error())
    }
}
