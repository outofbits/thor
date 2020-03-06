package monitor

import (
    log "github.com/sirupsen/logrus"
    "github.com/sobitada/go-cardano"
    jor "github.com/sobitada/go-jormungandr/api"
    "github.com/sobitada/thor/utils"
    "math/big"
    "time"
)

type ActionContext struct {
    TimeSettings         *cardano.TimeSettings
    BlockHeightMap       map[string]*big.Int
    MaximumBlockHeight   *big.Int
    UpToDateNodes        []string
    LastNodeStatisticMap map[string]jor.NodeStatistic
}

type Action interface {
    execute(nodes []Node, context ActionContext)
}

type ShutDownWithBlockLagAction struct{}

func (action ShutDownWithBlockLagAction) execute(nodes []Node, context ActionContext) {
    for p := range nodes {
        peer := nodes[p]
        if peer.MaxBlockLag == 0 { // ignore nodes that have not set a MaxInt block lag.
            continue
        }
        nodeStats, found := context.LastNodeStatisticMap[peer.Name]
        if !found || nodeStats.UpTime <= peer.WarmUpTime { // give the node some time to warm up
            continue
        }
        peerBlockHeight, found := context.BlockHeightMap[peer.Name]
        if found {
            lag := new(big.Int).Sub(context.MaximumBlockHeight, peerBlockHeight)
            if lag.Cmp(new(big.Int).SetUint64(peer.MaxBlockLag)) >= 0 {
                log.Warnf("[%s] Pool has fallen behind %v blocks.", peer.Name, lag.String())
                go ShutDownNode(peer)
            }
        }
    }
}

type ShutDownWhenStuck struct{}

func (action ShutDownWhenStuck) execute(nodes []Node, context ActionContext) {
    if context.TimeSettings != nil {
        for p := range nodes {
            peer := nodes[p]
            lastBlock, found := context.LastNodeStatisticMap[peer.Name]
            if peer.MaxTimeSinceLastBlock <= 0 { // ignore nodes that have not set a MaxInt duration.
                continue
            }
            nodeStats, found := context.LastNodeStatisticMap[peer.Name]
            if !found || nodeStats.UpTime <= peer.WarmUpTime { // give the node some time to warm up
                continue
            }
            if found {
                mostRecentBlockDate := cardano.MakeFullSlotDate(lastBlock.LastBlockDate, *context.TimeSettings)
                diff := time.Now().Sub(mostRecentBlockDate.GetEndDateTime())
                if diff > peer.MaxTimeSinceLastBlock {
                    log.Warnf("[%s] Most recent received block is %v old.", peer.Name, utils.GetHumanReadableUpTime(diff))
                    go ShutDownNode(peer)
                }
            }
        }
    }
}