package pooltool

import (
    "github.com/boltdb/bolt"
    "github.com/sobitada/go-cardano"
    jor "github.com/sobitada/go-jormungandr/api"
    "github.com/sobitada/thor/monitor"
)

// Pool Tool object, which specifies the user
// id, pool id and the hash of the genesis
// block.
type PoolTool struct {
    poolID         string
    userID         string
    genesisHash    string
    tipUpdate      *tipUpdate
    scheduleUpdate *scheduleUpdate
}

// constructs a new pool tool with the given configuration.
func GetPoolTool(mon *monitor.NodeMonitor, watchDog *monitor.ScheduleWatchDog, timeSettings *cardano.TimeSettings, db *bolt.DB,
    poolID string, userID string, genesisHash string) *PoolTool {
    // tip
    tipListener := make(chan map[string]jor.NodeStatistic)
    mon.ListenerManager.RegisterNodeStatisticListener(tipListener)
    // schedule
    scheduleListener := make(chan []jor.LeaderAssignment)
    watchDog.RegisterListener(scheduleListener)
    return &PoolTool{
        poolID:      poolID,
        userID:      userID,
        genesisHash: genesisHash,
        tipUpdate: &tipUpdate{
            latestTip:        nil,
            latestTipChannel: tipListener,
        },
        scheduleUpdate: &scheduleUpdate{
            db:             db,
            timeSettings:   timeSettings,
            latestSchedule: scheduleListener,
        },
    }
}

// starts the pool tool update client.
func (poolTool *PoolTool) Start() {
    go poolTool.startTipUpdating()
    go poolTool.startScheduleUpdating()
}
