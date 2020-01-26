package pooltool

import (
    log "github.com/sirupsen/logrus"
    jor "github.com/sobitada/go-jormungandr/api"
    "github.com/sobitada/thor/monitor"
    "github.com/sobitada/thor/utils"
    "math/big"
    "net/http"
    "net/url"
    "time"
)

const poolToolTipURL string = "https://tamoq3vkbl.execute-api.us-west-2.amazonaws.com/prod/sharemytip"

// team of Pool Tool asked to keep rate low.
const tipPostLimit time.Duration = 30 * time.Second

// Pool Tool object, which specifies the user
// id, pool id and the hash of the genesis
// block.
type PoolTool struct {
    poolID           string
    userID           string
    genesisHash      string
    latestTip        *big.Int
    latestTipChannel chan map[string]jor.NodeStatistic
}

// constructs a new pool tool with the given configuration.
func GetPoolTool(mon *monitor.NodeMonitor, poolID string, userID string, genesisHash string) *PoolTool {
    listener := make(chan map[string]jor.NodeStatistic)
    mon.ListenerManager.RegisterNodeStatisticListener(listener)
    return &PoolTool{poolID: poolID, userID: userID, genesisHash: genesisHash, latestTip: nil, latestTipChannel: listener}
}

// informs pool tool about the latest block height.
func (poolTool *PoolTool) PushLatestTip(tip *big.Int) {
    poolTool.latestTip = tip
}

func (poolTool *PoolTool) updateTip() {
    for ; ; {
        latestBlockStats := <-poolTool.latestTipChannel
        // compute max
        blockHeightMap := make(map[string]*big.Int)
        for name, nodeStats := range latestBlockStats {
            blockHeightMap[name] = nodeStats.LastBlockHeight
        }
        maxHeight, _ := utils.MaxInt(blockHeightMap)
        // update
        poolTool.latestTip = maxHeight
    }
}

// starts the pool tool update client.
func (poolTool *PoolTool) Start() {
    go poolTool.updateTip()
    for ; ; {
        if poolTool.latestTip != nil && poolTool.latestTip.Cmp(new(big.Int).SetUint64(0)) > 0 {
            err := poolTool.postLatestTip(poolTool.latestTip)
            if err != nil {
                log.Warnf("Could not post to pool tool. %v", err.Error())
            }
        }
        time.Sleep(tipPostLimit)
    }
}

// posts the given block height to the pool tool API.
func (poolTool *PoolTool) postLatestTip(tip *big.Int) error {
    u, err := url.Parse(poolToolTipURL)
    if err == nil {
        q := u.Query()
        q.Set("poolid", poolTool.poolID)
        q.Set("userid", poolTool.userID)
        q.Set("genesispref", poolTool.genesisHash)
        q.Set("mytip", tip.String())
        u.RawQuery = q.Encode()
        response, err := http.Get(u.String())
        if err == nil {
            if response.StatusCode == 200 {
                return nil
            } else {
                return poolToolAPIException{URL: poolToolTipURL, StatusCode: response.StatusCode, Reason: response.Status}
            }
        }
        return err
    }
    return err
}
