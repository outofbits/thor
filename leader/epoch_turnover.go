package leader

import (
    log "github.com/sirupsen/logrus"
    "github.com/sobitada/go-cardano"
    "github.com/sobitada/go-jormungandr/api"
    "github.com/sobitada/thor/monitor"
    "github.com/sobitada/thor/utils"
    "math/big"
    "time"
)

func nextEpochStart(slotDate *cardano.FullSlotDate, timeSettings cardano.TimeSettings) *cardano.FullSlotDate {
    epochDate, _ := cardano.FullSlotDateFrom(new(big.Int).Add(slotDate.GetEpoch(), new(big.Int).SetInt64(1)),
        new(big.Int).SetInt64(0), timeSettings)
    return epochDate
}

func promoteNode(node monitor.Node, cert api.LeaderCertificate, nextEpoch *cardano.FullSlotDate,
    settings cardano.TimeSettings) {
    _, err := node.API.PostLeader(cert)
    if err != nil {
        log.Warnf("[TURNOVER] Could not promote node %v. %v", node.Name, err.Error())
        if !time.Now().After(nextEpoch.GetStartDateTime().Add(-1 * settings.SlotDuration)) {
            diff := nextEpoch.GetStartDateTime().Add(-1 * settings.SlotDuration).Sub(time.Now())
            time.Sleep(utils.MaxDuration(diff, 5*settings.SlotDuration))
            go promoteNode(node, cert, nextEpoch, settings)
        }
    }
}

func (jury *Jury) turnOverHandling() {
    for ; ; {
        currentSlotDate, _ := jury.settings.TimeSettings.GetSlotDateFor(time.Now())
        // get time for turn over.
        nextEpoch := nextEpochStart(currentSlotDate, *jury.settings.TimeSettings)
        schedule, found := jury.watchDog.GetScheduleFor(currentSlotDate.GetEpoch())
        leaderPromotionDate := nextEpoch.GetStartDateTime().Add(-time.Duration(jury.settings.PreEpochTurnOverExclusionSlots.Int64()) * jury.settings.TimeSettings.SlotDuration)
        // get last assignment in this epoch
        if found && schedule != nil && len(schedule) > 0 {
            lastAssignment := schedule[len(schedule)-1]
            afterLastAssignmentSlotDate, _ := cardano.FullSlotDateFrom(lastAssignment.ScheduleBlockDate.GetEpoch(),
                lastAssignment.ScheduleBlockDate.GetSlot(), *jury.settings.TimeSettings)
            afterLastAssignmentDate := afterLastAssignmentSlotDate.GetEndDateTime()
            if afterLastAssignmentDate.After(leaderPromotionDate) {
                leaderPromotionDate = afterLastAssignmentDate.Add(500 * time.Millisecond)
            }
        }
        waitTime := leaderPromotionDate.Sub(time.Now())
        log.Infof("[TURNOVER] Waiting %s for handling turn over.", utils.GetHumanReadableUpTime(waitTime))
        if waitTime > 0 {
            time.Sleep(waitTime)
        }
        // promote all nodes to leader
        for _, node := range jury.nodes {
            go promoteNode(node, jury.cert, nextEpoch, *jury.settings.TimeSettings)
        }
        waitTime = nextEpoch.GetEndDateTime().Add(2 * jury.settings.TimeSettings.SlotDuration).Sub(time.Now())
        if waitTime > 0 {
            time.Sleep(waitTime)
        }
        // do sanity check
        jury.sanityCheck()
        // waiting a bit for new turn over handling check.
        time.Sleep(time.Duration(jury.settings.EpochTurnOverExclusionSlots.Int64()) * jury.settings.TimeSettings.SlotDuration)
    }
}
