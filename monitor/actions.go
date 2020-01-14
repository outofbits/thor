package monitor

import (
    "fmt"
    log "github.com/sirupsen/logrus"
    "github.com/sobitada/go-cardano"
    jor "github.com/sobitada/go-jormungandr/api"
    "math/big"
    "net/smtp"
    "strings"
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
        if peer.MaxBlockLag == 0 { // ignore nodes that have not set a max block lag.
            continue
        }
        peerBlockHeight, found := context.BlockHeightMap[peer.Name]
        if found {
            lag := new(big.Int).Sub(context.MaximumBlockHeight, peerBlockHeight)
            if lag.Cmp(new(big.Int).SetUint64(peer.MaxBlockLag)) >= 0 {
                log.Warnf("[%s] Pool has fallen behind %v blocks.", peer.Name, lag.String())
                go shutDownNode(peer)
            }
        }
    }
}

type EmailActionConfig struct {
    SourceAddress        string
    DestinationAddresses []string
    ServerURL            string
    Authentication       smtp.Auth
}

type ReportBlockLagPerEmailAction struct {
    Config EmailActionConfig
}

const latestBlockMessage string = `
Latest Block
------------
UpTime: %v
Received Blocks: %v
Received Transactions: %v
SlotDate: %v
Height: %v
Hash: %v
`

func getLatestBlockMessage(nodeStatistic jor.NodeStatistic) string {
    return fmt.Sprintf(latestBlockMessage, nodeStatistic.UpTime.String(), nodeStatistic.ReceivedBlocks.String(),
        nodeStatistic.ReceivedTransactions.String(), nodeStatistic.LastBlockDate.String(),
        nodeStatistic.LastBlockHeight, nodeStatistic.LastBlockHash)
}

const blockLagMessage string = `
Node '%v' has fallen behind %v blocks.

Timestamp: %v

%v
`

func (action ReportBlockLagPerEmailAction) execute(nodes []Node, context ActionContext) {
    for p := range nodes {
        peer := nodes[p]
        peerBlockHeight, found := context.BlockHeightMap[peer.Name]
        if peer.MaxBlockLag == 0 { // ignore nodes that have not set a max block lag.
            continue
        }
        if found {
            lag := new(big.Int).Sub(context.MaximumBlockHeight, peerBlockHeight)
            if lag.Cmp(new(big.Int).SetUint64(peer.MaxBlockLag)) >= 0 {
                go sendEmailReport(action.Config, fmt.Sprintf("[THOR][%v] Report of Block Lag.", peer.Name),
                    fmt.Sprintf(blockLagMessage, peer.Name, lag, time.Now(), getLatestBlockMessage(context.LastNodeStatisticMap[peer.Name])))
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
            if peer.MaxTimeSinceLastBlock <= 0 { // ignore nodes that have not set a max duration.
                continue
            }
            if found {
                mostRecentBlockDate := cardano.MakeFullSlotDate(lastBlock.LastBlockDate, *context.TimeSettings)
                diff := time.Now().Sub(mostRecentBlockDate.GetEndDateTime())
                if diff > peer.MaxTimeSinceLastBlock {
                    log.Warnf("[%s] Most recent received block is %v old.", peer.Name, getHumanReadableUpTime(diff))
                    go shutDownNode(peer)
                }
            }
        }
    }
}

type ReportStuckPerEmailAction struct {
    Config EmailActionConfig
}

const stuckMessage string = `
Node '%v' most recent block was computed at %v.

Timestamp: %v

%v
`

func (action ReportStuckPerEmailAction) execute(nodes []Node, context ActionContext) {
    if context.TimeSettings != nil {
        for p := range nodes {
            peer := nodes[p]
            lastBlock, found := context.LastNodeStatisticMap[peer.Name]
            if peer.MaxTimeSinceLastBlock <= 0 { // ignore nodes that have not set a max duration.
                continue
            }
            if found {
                mostRecentBlockDate := cardano.MakeFullSlotDate(lastBlock.LastBlockDate, *context.TimeSettings)
                diff := time.Now().Sub(mostRecentBlockDate.GetEndDateTime())
                if diff > peer.MaxTimeSinceLastBlock {
                    sendEmailReport(action.Config, fmt.Sprintf("[THOR][%v] Report Blockchain Stuck.", peer.Name),
                        fmt.Sprintf(stuckMessage, peer.Name, mostRecentBlockDate.GetStartDateTime().String(),
                            time.Now().String(), getLatestBlockMessage(context.LastNodeStatisticMap[peer.Name])))
                }
            }
        }
    }
}

func sendEmailReport(config EmailActionConfig, subject string, message string) {
    msg := []byte(fmt.Sprintf("To: %v\r\nSubject: %v\r\n\r\n%v\r\n", strings.Join(config.DestinationAddresses, ";"), subject, message))
    err := smtp.SendMail(config.ServerURL, config.Authentication, config.SourceAddress, config.DestinationAddresses, msg)
    if err != nil {
        log.Errorf("Could not send email. %v", err.Error())
    }
}
