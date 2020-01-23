package monitor

import (
    log "github.com/sirupsen/logrus"
    "github.com/sobitada/go-jormungandr/api"
    "time"
)

// sanity check tries to correct fail overs and potential flaws in this
// program as well as in Jormungandr. It checks whether only one node is
// promoted to a leader. It is important to avoid adversarial forks,
// because creating such a fork causes public shame! shame! shaming and
// blacklisting.
func (jury *Jury) sanityCheck(scheduleChannel chan []api.LeaderAssignment) {
    for ; ; {
        assignments := <-scheduleChannel
        currentSlotDate, err := jury.settings.TimeSettings.GetSlotDateFor(time.Now())
        if err != nil {
            log.Fatalf("[LEADER JURY] Sanity check loop panicked: %v", err.Error())
            time.Sleep(30 * time.Minute)
            continue
        }
        nextAssignments := api.FilterLeaderLogsBefore(time.Now().Add(2*time.Minute),
            api.SortLeaderLogsByScheduleTime(api.FilterForLeaderLogsInEpoch(currentSlotDate.GetEpoch(), assignments)))
        log.Debugf("[LEADER JURY] Started sanity check for %v assignments ahead. ", len(nextAssignments))
        for i := 0; i < len(nextAssignments); i++ {
            waitDuration := nextAssignments[i].ScheduleTime.Sub(time.Now()) - 1*time.Minute
            if waitDuration > 0 { // no sanity check between slots that are too close to each other.
                log.Infof("[LEADER JURY] Waiting %v for the next sanity check.", waitDuration.String())
                time.Sleep(waitDuration)
                log.Infof("[LEADER JURY] Sanity check before assignment %v.", nextAssignments[i].ScheduleTime)
                // do sanity checking
                jury.leaderMutex.Lock()
                for name, node := range jury.nodes {
                    log.Infof("[LEADER JURY] Sanity check node %v.", name)
                    if jury.leader != nil && jury.leader.name == name {
                        jury.sanityCheckLeaderNode(node)
                    } else {
                        jury.sanityCheckPassiveNode(node)
                    }
                }
                jury.leaderMutex.Unlock()
            }
        }
        time.Sleep(1 * time.Minute)
    }
}

// check whether a passive node is not unintentionally promoted to
// a leader node. If so, demote this node in the sanity check.
func (jury *Jury) sanityCheckPassiveNode(node Node) {
    leaderIDs, err := node.API.GetRegisteredLeaders()
    if err == nil {
        if len(leaderIDs) > 0 {
            log.Warnf("[LEADER JURY] Node %v is in leader mode while jury promoted other node.", node.Name)
            for i := range leaderIDs {
                demoteLeader(node, leaderIDs[i], 3)
            }
        }
    }
}

// check whether a leader node is promoted to a leader and if it
// also has not registered the leader twice (which can happens).
// correct the state of the leader node, if not in proper state.
func (jury *Jury) sanityCheckLeaderNode(node Node) {
    leaderIDs, err := node.API.GetRegisteredLeaders()
    if err == nil {
        leaderIDNumber := len(leaderIDs)
        if leaderIDNumber == 0 {
            log.Warnf("[LEADER JURY] Node %v is not promoted to leader node as expected.", node.Name)
            leaderID, err := node.API.PostLeader(jury.Cert)
            if err == nil {
                jury.leader = &currentLeader{name: node.Name, leaderID: leaderID}
                log.Infof("[LEADER JURY] Node %v is elected and has ID=%v", node.Name, leaderID)
            } else {
                log.Errorf("[LEADER JURY] Could not change to leader %v. %v", node.Name, err.Error())
            }
        } else if leaderIDNumber == 1 {
            log.Infof("[LEADER JURY] Node %v is leader as expected.", node.Name)
        } else {
            log.Warnf("[LEADER JURY] Node %v has more than one leader registered (%v).", node.Name, leaderIDNumber)
            for i := range leaderIDs {
                if leaderIDs[i] != jury.leader.leaderID {
                    demoteLeader(node, leaderIDs[i], 3)
                }
            }
        }
    }
}
