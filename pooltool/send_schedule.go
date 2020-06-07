package pooltool

import (
    "bytes"
    "encoding/base64"
    "encoding/json"
    "fmt"
    "github.com/boltdb/bolt"
    log "github.com/sirupsen/logrus"
    "github.com/sobitada/go-cardano"
    jor "github.com/sobitada/go-jormungandr/api"
    "golang.org/x/crypto/openpgp"
    "golang.org/x/crypto/openpgp/armor"
    "io/ioutil"
    "math/big"
    "math/rand"
    "net/http"
    "time"
)

const poolToolScheduleURL string = "https://api.pooltool.io/v0/sendlogs"

type scheduleUpdate struct {
    db             *bolt.DB
    timeSettings   *cardano.TimeSettings
    latestSchedule chan []jor.LeaderAssignment
}

// start the process of updating the schedule in each experienced epoch.
func (poolTool *PoolTool) startScheduleUpdating() {
    scheduleUpdate := poolTool.scheduleUpdate
    if scheduleUpdate != nil && scheduleUpdate.db != nil && scheduleUpdate.latestSchedule != nil {
        log.Info("[POOLTOOL] Start to update Pool Tool with our schedule.")
        for ; ; {
            schedule := <-scheduleUpdate.latestSchedule
            poolTool.updateSchedule(schedule)
        }
    } else {
        log.Warn("[POOLTOOL] No schedule update will be sent, due to misconfiguration.")
    }
}

// generates a 32 character long key encoded in base64.
func generateKey() string {
    var key = make([]byte, 32)
    rand.Read(key)
    return base64.StdEncoding.EncodeToString(key)
}

// transforms the given schedule into JSON and encrypts the JSOn string with the
// given key phrase. it returns the encrypted schedule as string, or an error, if
// the encryption failed.
func encryptSchedule(schedule []LeaderAssignment, key string) (string, error) {
    scheduleData, err := json.Marshal(schedule)
    log.Infof(">>JSON>> %v", string(scheduleData))
    if err == nil {
        armoredScheduleOut := bytes.NewBuffer(nil)
        w, err := armor.Encode(armoredScheduleOut, "PGP MESSAGE", nil)
        defer w.Close()
        if err == nil {
            plainTextWriter, err := openpgp.SymmetricallyEncrypt(w, []byte(key), nil, nil)
            if err == nil {
                _, err := plainTextWriter.Write(scheduleData)
                plainTextWriter.Close()
                if err == nil {
                    w.Close()
                    return string(armoredScheduleOut.Bytes()), nil
                } else {
                    return "", err
                }
            } else {
                return "", err
            }
        } else {
            return "", err
        }
    } else {
        return "", err
    }
}

// analysis the given schedule, and checks whether an update is necessary. if so, the
// update will be issued.
func (poolTool *PoolTool) updateSchedule(schedule []jor.LeaderAssignment) {
    currentSlotDate, _ := poolTool.scheduleUpdate.timeSettings.GetSlotDateFor(time.Now())
    currentEpoch := currentSlotDate.GetEpoch()
    currentKeyPhrase := poolTool.scheduleUpdate.getKey(currentEpoch)
    if currentKeyPhrase == "" || true {
        // issue the update
        schedule = jor.GetLeaderLogsInEpoch(currentEpoch, schedule)
        if len(schedule) > 0 {
            key := generateKey()
            data, err := encryptSchedule(transformAssignments(schedule), key)
            log.Infof(">>Key>> %v", key)
            log.Infof(">>Encrypted>> %v", data)
            if err == nil {
                previousEpoch := new(big.Int).Sub(currentEpoch, new(big.Int).SetInt64(1))
                previousEpochKey := poolTool.scheduleUpdate.getKey(previousEpoch)
                err := poolTool.postSchedule(currentEpoch, len(schedule), data, previousEpochKey)
                if err == nil {
                    err := poolTool.scheduleUpdate.storeKey(currentEpoch, key)
                    if err != nil {
                        log.Errorf("[POOLTOOL] Could not persist the key for epoch '%v'. %v",
                            currentEpoch.String(), err.Error())
                    }
                } else {
                    log.Errorf("[POOLTOOL] Could not send schedule update to Pool Tool. %v",
                        err.Error())
                }
            } else {
                log.Errorf("[POOLTOOL] Could not encrypt the schedule for epoch '%v'. %v",
                    currentEpoch.String(), err.Error())
            }
        }
    } else {
        log.Warnf("[POOLTOOL] Schedule has already been updated for epoch '%v'.",
            currentEpoch.String())
    }
}

// payload for sending a schedule update to Pool Tool.
type postSchedulePayload struct {
    CurrentEpoch     int64  `json:"currentepoch"`
    PoolID           string `json:"poolid"`
    UserID           string `json:"userid"`
    GenesisPref      string `json:"genesispref"`
    AssignedSlots    int    `json:"assigned_slots"`
    EncryptedSlots   string `json:"encrypted_slots"`
    PreviousEpochKey string `json:"previous_epoch_key,omitempty"`
}

// a leader assignment as expected by Pool Tool.
type LeaderAssignment struct {
    CreatedAtTime   string  `json:"created_at_time"`
    ScheduledAtTime string  `json:"scheduled_at_time"`
    ScheduledAtDate string  `json:"scheduled_at_date"`
    WakeAtTime      *string `json:"wake_at_time"`
    FinishedAtTime  *string `json:"finished_at_time"`
    Status          string  `json:"status"`
    EnclaveLeaderID int     `json:"enclave_leader_id"`
}

// transforms the given leader assignments passed by the Jormungandr wrapper into
// a format that is expected by Pool Tool.
func transformAssignments(assignments []jor.LeaderAssignment) []LeaderAssignment {
    transformedAssignments := make([]LeaderAssignment, len(assignments))
    for i, entry := range assignments {
        transformedAssignment := LeaderAssignment{
            CreatedAtTime:   entry.CreationTime.Format("2006-01-02T15:04:05.999999999-07:00"),
            ScheduledAtTime: entry.ScheduleTime.Format("2006-01-02T15:04:05.999999999-07:00"),
            ScheduledAtDate: entry.ScheduleBlockDate.String(),
            Status:          "Pending",
            EnclaveLeaderID: 1,
        }
        transformedAssignments[i] = transformedAssignment
    }
    return transformedAssignments
}

// response payload from Pool Tool for the API method
// accepting schedule update.
type postScheduleResponse struct {
    Success bool                       `json:"success"`
    Message map[string]json.RawMessage `json:"message"`
}

// posts the schedule information to Pool Tool. It requires the epoch for which the schedule
// shall be updated, the number of assigned slots in the epoch and the encrypted JSON serialization
// of the schedule. If there has been an update for the previous epoch, then the key for encrypting
// the data for the previous epoch must be specified. However, this field is optional.
func (poolTool *PoolTool) postSchedule(epoch *big.Int, slotsNumber int, encryptedSchedule string,
    previousEpochKey string) error {
    payload := &postSchedulePayload{
        CurrentEpoch:     epoch.Int64(),
        PoolID:           poolTool.poolID,
        UserID:           poolTool.userID,
        GenesisPref:      poolTool.genesisHash,
        AssignedSlots:    slotsNumber,
        EncryptedSlots:   encryptedSchedule,
        PreviousEpochKey: previousEpochKey,
    }
    payloadBytes, err := json.Marshal(payload)
    if err == nil {
        reader := bytes.NewReader(payloadBytes)
        response, err := http.Post(poolToolScheduleURL, "application/json", reader)
        if err == nil {
            if response.StatusCode == 200 {
                responseData, err := ioutil.ReadAll(response.Body)
                if err == nil {
                    if responseData != nil {
                        var responseJSON postScheduleResponse
                        err := json.Unmarshal(responseData, &responseJSON)
                        if err == nil {
                            if responseJSON.Success {
                                return nil
                            } else {
                                var reason string = ""
                                reasonRaw, found := responseJSON.Message["error"]
                                if found {
                                    _ = json.Unmarshal(reasonRaw, &reason)
                                }
                                if reason == "We were unable to parse the decrypted json data.  That either means we were unable to decrypt it, or the json is not valid" {
                                    log.Warn("Could not post the schedule to Pool Tool, because the information sent in the previous epoch was malformed. Trying without passing key phrase about previous epoch.")
                                    return poolTool.postSchedule(epoch, slotsNumber, encryptedSchedule, "")
                                }
                                return poolToolAPIException{
                                    URL:        poolToolScheduleURL,
                                    StatusCode: response.StatusCode,
                                    Reason:     fmt.Sprintf("Request was rejected, due to: %v", reason),
                                }
                            }
                        } else {
                            return poolToolAPIException{
                                URL:        poolToolScheduleURL,
                                StatusCode: response.StatusCode,
                                Reason:     fmt.Sprintf("Could not serialize the JSON body.%v", err.Error()),
                            }
                        }
                    } else {
                        return poolToolAPIException{
                            URL:        poolToolScheduleURL,
                            StatusCode: response.StatusCode,
                            Reason:     "Could not read the body of the response.",
                        }
                    }
                } else {
                    return poolToolAPIException{
                        URL:        poolToolScheduleURL,
                        StatusCode: response.StatusCode,
                        Reason:     fmt.Sprintf("Could not read the body of the response. %v", err.Error()),
                    }
                }
            } else {
                return poolToolAPIException{
                    URL:        poolToolScheduleURL,
                    StatusCode: response.StatusCode,
                    Reason:     response.Status,
                }
            }
        }
        return err
    }
    return err
}

// stores the given key phrase under the given epoch into the key,value db.
func (scheduleUpdate *scheduleUpdate) storeKey(epoch *big.Int, key string) error {
    return scheduleUpdate.db.Update(func(tx *bolt.Tx) error {
        b, err := tx.CreateBucketIfNotExists([]byte("schedule-epoch-keys"))
        if err == nil && b != nil {
            return b.Put([]byte(epoch.String()), []byte(key))
        }
        return err
    })
}

// gets the key for the given epoch, if it can be found, or an empty string otherwise.
func (scheduleUpdate *scheduleUpdate) getKey(epoch *big.Int) string {
    initialKey := ""
    keyPtr := &initialKey
    _ = scheduleUpdate.db.View(func(tx *bolt.Tx) error {
        b := tx.Bucket([]byte("schedule-epoch-keys"))
        if b != nil {
            keyData := b.Get([]byte(epoch.String()))
            if keyData != nil {
                keyStr := string(keyData)
                keyPtr = &keyStr
            }
        }
        return nil
    })
    return *keyPtr
}
