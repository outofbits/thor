package monitor

import (
    log "github.com/sirupsen/logrus"
    "github.com/sobitada/go-cardano"
    "github.com/sobitada/go-jormungandr/api"
    "github.com/sobitada/thor/utils"
    "math/big"
    "sync"
    "time"
)

type ScheduleWatchDog struct {
    nodes        []Node
    scheduleMap  map[string][]api.LeaderAssignment
    timeSettings *cardano.TimeSettings
    mutex        *sync.RWMutex
    listeners    listener
}

type listener struct {
    list  []chan []api.LeaderAssignment
    mutex *sync.Mutex
}

func NewScheduleWatchDog(nodes []Node, timeSettings *cardano.TimeSettings) *ScheduleWatchDog {
    scheduleMap := make(map[string][]api.LeaderAssignment)
    listenerList := make([]chan []api.LeaderAssignment, 0)
    return &ScheduleWatchDog{
        nodes:        nodes,
        scheduleMap:  scheduleMap,
        timeSettings: timeSettings,
        mutex:        &sync.RWMutex{},
        listeners: listener{
            list:  listenerList,
            mutex: &sync.Mutex{},
        },
    }
}

func getCurrentSchedule(epoch *big.Int, node Node) []api.LeaderAssignment {
    schedule, err := node.API.GetLeadersSchedule()
    if err == nil && schedule != nil {
        return api.FilterForLeaderLogsInEpoch(epoch, api.SortLeaderLogsByScheduleTime(schedule))
    }
    return schedule
}

func nextEpochStart(slotDate *cardano.FullSlotDate, timeSettings cardano.TimeSettings) *cardano.FullSlotDate {
    epochDate, _ := cardano.FullSlotDateFrom(new(big.Int).Add(slotDate.GetEpoch(), new(big.Int).SetInt64(1)),
        new(big.Int).SetInt64(2), timeSettings)
    return epochDate
}

func (watchDog *ScheduleWatchDog) RegisterListener(listener chan []api.LeaderAssignment) {
    watchDog.listeners.mutex.Lock()
    defer watchDog.listeners.mutex.Unlock()
    watchDog.listeners.list = append(watchDog.listeners.list, listener)
}

func (watchDog *ScheduleWatchDog) GetScheduleFor(epoch *big.Int) ([]api.LeaderAssignment, bool) {
    watchDog.mutex.RLock()
    defer watchDog.mutex.RUnlock()
    schedule, found := watchDog.scheduleMap[epoch.String()]
    return schedule, found
}

func (watchDog *ScheduleWatchDog) informListenerAboutSchedule(schedule []api.LeaderAssignment) {
    watchDog.listeners.mutex.Lock()
    defer watchDog.listeners.mutex.Unlock()
    for i := range watchDog.listeners.list {
        watchDog.listeners.list[i] <- schedule
    }
}

func (watchDog *ScheduleWatchDog) Watch() {
    log.Info("[SCHEDULE] Starting to watch the schedule.")
    var next time.Duration = 0
    for ; ; {
        time.Sleep(next)
        fetchSchedule := true
        currentSlotDate, _ := watchDog.timeSettings.GetSlotDateFor(time.Now())
        watchDog.mutex.RLock()
        schedule, found := watchDog.scheduleMap[currentSlotDate.GetEpoch().String()]
        if found && schedule != nil {
            if len(schedule) > 0 {
                fetchSchedule = true
            }
        }
        watchDog.mutex.RUnlock()
        if !fetchSchedule {
            log.Infof("[SCHEDULE] The schedule has already been fetched for epoch %v. (%v) entries.",
                currentSlotDate.GetEpoch().String(), len(schedule))
            next = nextEpochStart(currentSlotDate, *watchDog.timeSettings).GetEndDateTime().Sub(time.Now())
            continue
        } else {
            log.Infof("[SCHEDULE] The schedule for epoch %v will be fetched.",
                currentSlotDate.GetEpoch().String())
            var newSchedule []api.LeaderAssignment = nil
            for n := range watchDog.nodes {
                newSchedule = getCurrentSchedule(currentSlotDate.GetEpoch(), watchDog.nodes[n])
                if newSchedule != nil && len(newSchedule) > 0 {
                    break
                }
            }
            if newSchedule != nil && len(newSchedule) > 0 {
                watchDog.mutex.Lock()
                watchDog.scheduleMap[currentSlotDate.GetEpoch().String()] = newSchedule
                watchDog.mutex.Unlock()
                watchDog.informListenerAboutSchedule(newSchedule)
                log.Infof("[SCHEDULE] Watchdog fetched %v leader assignments for epoch %v.",
                    len(newSchedule), currentSlotDate.GetEpoch().String())
                next = nextEpochStart(currentSlotDate, *watchDog.timeSettings).GetEndDateTime().Sub(time.Now())
            } else {
                next = 10 * time.Minute
            }
        }
        log.Infof("[SCHEDULE] Waiting %v for next check.", utils.GetHumanReadableUpTime(next))
    }
}
