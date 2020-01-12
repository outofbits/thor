package pooltool

import (
    log "github.com/sirupsen/logrus"
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
    poolID      string
    userID      string
    genesisHash string
    latestTip   *big.Int
}

// constructs a new pool tool with the given configuration.
func GetPoolTool(poolID string, userID string, genesisHash string) *PoolTool {
    return &PoolTool{poolID: poolID, userID: userID, genesisHash: genesisHash, latestTip: nil}
}

// informs pool tool about the latest block height.
func (poolTool *PoolTool) PushLatestTip(tip *big.Int) {
    poolTool.latestTip = tip
}

// starts the pool tool update client.
func (poolTool *PoolTool) Start() {
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
