package monitor

import (
    "encoding/json"
    "errors"
    "github.com/boltdb/bolt"
    log "github.com/sirupsen/logrus"
    "github.com/sobitada/go-cardano"
    "github.com/sobitada/go-jormungandr/api"
    "github.com/sobitada/thor/threading"
    "github.com/sobitada/thor/utils"
    "math/big"
    "sync"
    "time"
)

type ScheduleWatchDog struct {
    nodes             []Node
    db                *bolt.DB
    viableLeaderNodes viableLeaderNodes
    scheduleMap       map[string][]api.LeaderAssignment
    timeSettings      *cardano.TimeSettings
    mutex             *sync.RWMutex
    listeners         listener
}

type viableLeaderNodes struct {
    epochMap map[string][]string
    mutex    *sync.Mutex
}

type listener struct {
    list  []chan []api.LeaderAssignment
    mutex *sync.Mutex
}

// creates a new schedule watchdog for the given nodes and time
// settings of the block chain. both are required.
func NewScheduleWatchDog(nodes []Node, timeSettings *cardano.TimeSettings, db *bolt.DB) *ScheduleWatchDog {
    scheduleMap := make(map[string][]api.LeaderAssignment)
    listenerList := make([]chan []api.LeaderAssignment, 0)
    err := db.Update(func(tx *bolt.Tx) error {
        _, err := tx.CreateBucketIfNotExists([]byte("schedule"))
        return err
    })
    if err != nil {
        log.Fatal(err.Error())
    }
    return &ScheduleWatchDog{
        nodes:        nodes,
        scheduleMap:  scheduleMap,
        timeSettings: timeSettings,
        mutex:        &sync.RWMutex{},
        viableLeaderNodes: viableLeaderNodes{
            epochMap: map[string][]string{},
            mutex:    &sync.Mutex{},
        },
        db: db,
        listeners: listener{
            list:  listenerList,
            mutex: &sync.Mutex{},
        },
    }
}

// registers a listener that will be informed about a new schedule.
func (watchDog *ScheduleWatchDog) RegisterListener(listener chan []api.LeaderAssignment) {
    watchDog.listeners.mutex.Lock()
    defer watchDog.listeners.mutex.Unlock()
    watchDog.listeners.list = append(watchDog.listeners.list, listener)
}

// gets the schedule for the given epoch and boolean value indicating, whether
// the schedule has been fetched.
func (watchDog *ScheduleWatchDog) GetScheduleFor(epoch *big.Int) ([]api.LeaderAssignment, bool) {
    watchDog.mutex.RLock()
    defer watchDog.mutex.RUnlock()
    schedule, found := watchDog.scheduleMap[epoch.String()]
    return schedule, found
}

// gets viable leader nodes, i.e. nodes that have computed the
// identical leader schedule.
func (watchDog *ScheduleWatchDog) GetViableLeaderNodes() []string {
    currentSlotDate, _ := watchDog.timeSettings.GetSlotDateFor(time.Now())
    watchDog.viableLeaderNodes.mutex.Lock()
    defer watchDog.viableLeaderNodes.mutex.Unlock()
    return watchDog.viableLeaderNodes.epochMap[currentSlotDate.GetEpoch().String()]
}

func nextEpochStart(slotDate *cardano.FullSlotDate, timeSettings cardano.TimeSettings) *cardano.FullSlotDate {
    epochDate, _ := cardano.FullSlotDateFrom(new(big.Int).Add(slotDate.GetEpoch(), new(big.Int).SetInt64(1)),
        new(big.Int).SetInt64(2), timeSettings)
    return epochDate
}

func (watchDog *ScheduleWatchDog) checkViability(node Node, epoch *big.Int, schedule []api.LeaderAssignment) {
    if schedule != nil && len(schedule) > 0 {
        for ; ; {
            currentSlotDate, _ := watchDog.timeSettings.GetSlotDateFor(time.Now())
            if currentSlotDate.GetEpoch().Cmp(epoch) == 0 {
                break
            }
            newSchedule, err := node.API.GetLeadersSchedule()
            if err == nil {
                if newSchedule != nil && len(newSchedule) > 0 {
                    testTime := time.Now()
                    expectedSchedule := api.FilterLeaderLogsBefore(testTime, schedule)
                    newSchedule = api.FilterLeaderLogsBefore(testTime, newSchedule)
                    if len(expectedSchedule) == len(newSchedule) {
                        watchDog.viableLeaderNodes.mutex.Lock()
                        watchDog.viableLeaderNodes.epochMap[epoch.String()] = append(watchDog.viableLeaderNodes.epochMap[epoch.String()], node.Name)
                        watchDog.viableLeaderNodes.mutex.Unlock()
                        break
                    } else {
                        log.Warnf("[SCHEDULE] The leader schedule of node %v is of different length. Expected %v, but was %v.",
                            node.Name, len(newSchedule), len(schedule))
                    }
                } else {
                    log.Warnf("[SCHEDULE] Could not fetch schedule from %v.", node.Name)
                }
            } else {
                log.Warnf("[SCHEDULE] Could not fetch schedule from %v. %v", node.Name, err.Error())
            }
            time.Sleep(10 * time.Minute)
        }
    }
}

func getCurrentSchedule(epoch *big.Int, node Node) ([]api.LeaderAssignment, error) {
    schedule, err := node.API.GetLeadersSchedule()
    if err == nil && schedule != nil {
        return api.GetLeaderLogsOfLeader(1, api.GetLeaderLogsInEpoch(epoch, api.SortLeaderLogsByScheduleTime(schedule))), nil
    }
    return schedule, err
}

type sInput struct {
    epoch *big.Int
    node  Node
}

func fetchSchedule(input interface{}) threading.Response {
    sInput := input.(sInput)
    schedule, err := getCurrentSchedule(sInput.epoch, sInput.node)
    if err != nil {
        return threading.Response{
            Context: sInput.node,
            Error:   err,
        }
    } else {
        return threading.Response{
            Context: sInput.node,
            Data:    schedule,
        }
    }
}

// informs all the registered listener about the newly fetched schedule.
func (watchDog *ScheduleWatchDog) informListenerAboutSchedule(schedule []api.LeaderAssignment) {
    watchDog.listeners.mutex.Lock()
    defer watchDog.listeners.mutex.Unlock()
    for i := range watchDog.listeners.list {
        watchDog.listeners.list[i] <- schedule
    }
}

func (watchDog *ScheduleWatchDog) storeToDB(epoch *big.Int, schedule []api.LeaderAssignment) error {
    err := watchDog.db.Update(func(tx *bolt.Tx) error {
        b := tx.Bucket([]byte("schedule"))
        if b == nil {
            return errors.New("the bucket 'schedule' could not be found")
        } else {
            jsonData, err := json.Marshal(schedule)
            if err == nil && jsonData != nil {
                return b.Put([]byte(epoch.String()), jsonData)
            }
            return err
        }
    })
    return err
}

func (watchDog *ScheduleWatchDog) getFromDB(epoch *big.Int) ([]api.LeaderAssignment, error) {
    var storedSchedule *[]api.LeaderAssignment = nil
    err := watchDog.db.View(func(tx *bolt.Tx) error {
        b := tx.Bucket([]byte("schedule"))
        if b == nil {
            return errors.New("the bucket 'schedule' could not be found")
        } else {
            response := b.Get([]byte(epoch.String()))
            if response != nil {
                err := json.Unmarshal(response, storedSchedule)
                if err != nil {
                    return err
                }
            }
        }
        return nil
    })
    if err == nil && storedSchedule != nil {
        return *storedSchedule, nil
    }
    return nil, err
}

// watches for the schedules computed for epochs, and checks whether the
// leader candidates have computed the correct schedule.
func (watchDog *ScheduleWatchDog) Watch() {
    log.Info("[SCHEDULE] Starting to watch the schedule.")
    var next time.Duration = 0
    for ; ; {
        time.Sleep(next)
        shouldFetchSchedule := true
        currentSlotDate, _ := watchDog.timeSettings.GetSlotDateFor(time.Now())
        watchDog.mutex.RLock()
        schedule, found := watchDog.scheduleMap[currentSlotDate.GetEpoch().String()]
        if found && schedule != nil && len(schedule) > 0 {
            shouldFetchSchedule = false
        } else {
            storedSchedule, err := watchDog.getFromDB(currentSlotDate.GetEpoch())
            if err == nil && storedSchedule != nil {
                schedule = storedSchedule
                shouldFetchSchedule = false
            } else if err != nil {
                log.Errorf("[SCHEDULE] Could not fetch schedule from the DB. %v", err.Error())
            }
        }
        watchDog.mutex.RUnlock()
        if !shouldFetchSchedule {
            log.Infof("[SCHEDULE] The schedule has already been fetched for epoch %v. (%v) entries.",
                currentSlotDate.GetEpoch().String(), len(schedule))
            next = nextEpochStart(currentSlotDate, *watchDog.timeSettings).GetEndDateTime().Sub(time.Now())
            continue
        } else {
            log.Infof("[SCHEDULE] The schedule for epoch %v will be fetched.",
                currentSlotDate.GetEpoch().String())
            // make none of the leader candidates viable.
            watchDog.viableLeaderNodes.mutex.Lock()
            watchDog.viableLeaderNodes.epochMap[currentSlotDate.GetEpoch().String()] = []string{}
            watchDog.viableLeaderNodes.mutex.Unlock()
            // fetch the schedule
            var newSchedule []api.LeaderAssignment = nil
            inputs := make([]interface{}, len(watchDog.nodes))
            for i, node := range watchDog.nodes {
                inputs[i] = sInput{
                    node:  node,
                    epoch: currentSlotDate.GetEpoch(),
                }
            }
            viableLeaderNodes := make([]string, 0)
            noneViableLeaderNodes := make([]Node, 0)
            responses := threading.Complete(inputs, fetchSchedule)
            for _, response := range responses {
                node := response.Context.(Node)
                if response.Error == nil {
                    schedule := response.Data.([]api.LeaderAssignment)
                    if schedule != nil && len(schedule) > 0 {
                        if newSchedule == nil {
                            newSchedule = schedule
                            viableLeaderNodes = append(viableLeaderNodes, node.Name)
                        } else {
                            testTime := time.Now()
                            expectedSchedule := api.FilterLeaderLogsBefore(testTime, schedule)
                            newSchedule = api.FilterLeaderLogsBefore(testTime, newSchedule)
                            if len(expectedSchedule) == len(newSchedule) {
                                viableLeaderNodes = append(viableLeaderNodes, node.Name)
                            } else {
                                log.Warnf("[SCHEDULE] The leader schedule of node %v is of different length. Expected %v, but was %v.",
                                    node.Name, len(newSchedule), len(schedule))
                                noneViableLeaderNodes = append(noneViableLeaderNodes, node)
                            }
                        }
                    } else {
                        noneViableLeaderNodes = append(noneViableLeaderNodes, node)
                    }
                } else {
                    log.Warnf("[SCHEDULE] Could not fetch the leader schedule for %s.", node.Name)
                }
            }
            if newSchedule != nil && len(newSchedule) > 0 {
                // set schedule for this epoch.
                watchDog.mutex.Lock()
                watchDog.scheduleMap[currentSlotDate.GetEpoch().String()] = newSchedule
                watchDog.mutex.Unlock()
                // inform listeners about schedule.
                watchDog.informListenerAboutSchedule(newSchedule)
                // store to DB
                err := watchDog.storeToDB(currentSlotDate.GetEpoch(), schedule)
                if err != nil {
                    log.Errorf("[SCHEDULE] Could not store schedule for epoch %v. %v",
                        currentSlotDate.GetEpoch().String(), err.Error())
                }
                // set the viable leader nodes.
                watchDog.viableLeaderNodes.mutex.Lock()
                watchDog.viableLeaderNodes.epochMap[currentSlotDate.GetEpoch().String()] = viableLeaderNodes
                watchDog.viableLeaderNodes.mutex.Unlock()
                // check viability of non viable nodes periodically.
                for _, node := range noneViableLeaderNodes {
                    go watchDog.checkViability(node, currentSlotDate.GetEpoch(), newSchedule)
                }
                log.Infof("[SCHEDULE] Watchdog fetched %v leader assignments for epoch %v.",
                    len(newSchedule), currentSlotDate.GetEpoch().String())
                next = nextEpochStart(currentSlotDate, *watchDog.timeSettings).GetEndDateTime().Sub(time.Now())
            } else {
                if currentSlotDate.GetSlot().Cmp(new(big.Int).SetInt64(500)) <= 0 {
                    next = 50 * watchDog.timeSettings.SlotDuration
                } else {
                    next = utils.MinDuration(10*time.Minute,
                        nextEpochStart(currentSlotDate, *watchDog.timeSettings).GetEndDateTime().Sub(time.Now()))
                }
            }
        }
        log.Infof("[SCHEDULE] Waiting %v for next check.", utils.GetHumanReadableUpTime(next))
    }
}
