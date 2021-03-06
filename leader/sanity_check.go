package leader

import (
    log "github.com/sirupsen/logrus"
    "github.com/sobitada/go-jormungandr/api"
    "github.com/sobitada/thor/monitor"
    "github.com/sobitada/thor/threading"
    "github.com/sobitada/thor/utils"
    "time"
)

type NodeMode int

const (
    Demoted NodeMode = iota
    Promoted
)

type sanityInput struct {
    mode NodeMode
    jury *Jury
    node monitor.Node
}

func performSanityCheck(input interface{}) threading.Response {
    sInput := input.(sanityInput)
    log.Debugf("[LEADER JURY][SANITY CHECK] Start for node %v.", sInput.node.Name)
    if sInput.mode == Promoted {
        sInput.jury.sanityCheckLeaderNode(sInput.node)
    } else {
        sInput.jury.sanityCheckPassiveNode(sInput.node)
    }
    return threading.Response{
        Context: input,
        Data:    nil,
        Error:   nil,
    }
}

// sanity check tries to correct fail overs and potential flaws in this
// program as well as in Jormungandr. It checks whether only one node is
// promoted to a leader. It is important to avoid adversarial forks,
// because creating such a fork causes public shame! shame! shaming and
// blacklisting.
func (jury *Jury) startSanityChecks() {
    for ; ; {
        assignments := <-jury.scheduleChannel
        currentSlotDate, err := jury.settings.TimeSettings.GetSlotDateFor(time.Now())
        if err != nil {
            log.Fatalf("[LEADER JURY][SANITY CHECK] Loop panicked: %v", err.Error())
            time.Sleep(30 * time.Minute)
            continue
        }
        nextAssignments := api.FilterLeaderLogsBefore(time.Now().Add(2*time.Minute),
            api.SortLeaderLogsByScheduleTime(api.GetLeaderLogsInEpoch(currentSlotDate.GetEpoch(), assignments)))
        log.Debugf("[LEADER JURY][SANITY CHECK] Started sanity check for %v assignments ahead. ", len(nextAssignments))
        for i := 0; i < len(nextAssignments); i++ {
            waitDuration := nextAssignments[i].ScheduleTime.Sub(time.Now()) - 1*time.Minute
            if waitDuration > 0 { // no sanity check between slots that are too close to each other.
                log.Infof("[LEADER JURY][SANITY CHECK] Waiting %v for the next sanity check.",
                    utils.GetHumanReadableUpTime(waitDuration))
                time.Sleep(waitDuration)
                log.Infof("[LEADER JURY][SANITY CHECK] Check for assignment %v.", nextAssignments[i].ScheduleTime)
                // do sanity checking
                jury.leaderMutex.Lock()
                i := 0
                inputs := make([]interface{}, len(jury.nodes))
                for name, node := range jury.nodes {
                    if jury.leader != nil && jury.leader.name == name {
                        inputs[i] = sanityInput{node: node, mode: Promoted, jury: jury}
                    } else {
                        inputs[i] = sanityInput{node: node, mode: Demoted, jury: jury}
                    }
                    i++
                }
                threading.Complete(inputs, performSanityCheck)
                jury.leaderMutex.Unlock()
            }
        }
        time.Sleep(1 * time.Minute)
    }
}

// check the sanity of all nodes.
func (jury *Jury) sanityCheck() {
    jury.leaderMutex.Lock()
    for name, node := range jury.nodes {
        if jury.leader != nil && jury.leader.name == name {
            jury.sanityCheckLeaderNode(node)
        } else {
            jury.sanityCheckPassiveNode(node)
        }
    }
    jury.leaderMutex.Unlock()
}

// check whether a passive node is not unintentionally promoted to
// a leader node. If so, demote this node in the sanity check.
func (jury *Jury) sanityCheckPassiveNode(node monitor.Node) {
    leaderIDs, err := node.API.GetRegisteredLeaders()
    if err == nil {
        if len(leaderIDs) > 0 {
            log.Warnf("[LEADER JURY][SANITY CHECK][%v] In leader mode while jury promoted other node.", node.Name)
            for i := range leaderIDs {
                demoteLeader(node, leaderIDs[i], 3)
            }
        } else {
            log.Infof("[LEADER JURY][SANITY CHECK][%v] OK.", node.Name)
        }
    }
}

// check whether a leader node is promoted to a leader and if it
// also has not registered the leader twice (which can happens).
// correct the state of the leader node, if not in proper state.
func (jury *Jury) sanityCheckLeaderNode(node monitor.Node) {
    leaderIDs, err := node.API.GetRegisteredLeaders()
    if err == nil {
        leaderIDNumber := len(leaderIDs)
        if leaderIDNumber == 0 {
            log.Warnf("[LEADER JURY][SANITY CHECK][%v] Is not promoted to leader node as expected.", node.Name)
            leaderID, err := node.API.PostLeader(jury.cert)
            if err == nil {
                jury.leader = &currentLeader{name: node.Name, leaderID: leaderID}
                log.Infof("[LEADER JURY] Node %v is elected and has ID=%v", node.Name, leaderID)
                log.Infof("[LEADER JURY][SANITY CHECK][%v] OK.", node.Name)
            } else {
                log.Errorf("[LEADER JURY][SANITY CHECK][%v] Could not change to leader. %v", node.Name, err.Error())
            }
        } else if leaderIDNumber == 1 {
            log.Infof("[LEADER JURY][SANITY CHECK][%v] OK.", node.Name)
        } else {
            log.Warnf("[LEADER JURY][SANITY CHECK][%v] Has more than one leader registered (%v).", node.Name, leaderIDNumber)
            for i := range leaderIDs {
                if leaderIDs[i] != jury.leader.leaderID {
                    demoteLeader(node, leaderIDs[i], 3)
                }
            }
        }
    }
}
