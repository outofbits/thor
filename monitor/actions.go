package monitor

import (
    "fmt"
    log "github.com/sirupsen/logrus"
    jor "github.com/sobitada/go-jormungandr/api"
    "github.com/sobitada/go-jormungandr/cardano"
    "github.com/sobitada/thor/pooltool"
    "math/big"
    "net/smtp"
    "strings"
    "time"
)

type ActionContext struct {
    TimeSettings       *cardano.TimeSettings
    BlockHeightMap     map[string]*big.Int
    MaximumBlockHeight *big.Int
    UpToDateNodes      []string
    LastBlockMap       map[string]jor.NodeStatistic
}

type Action interface {
    execute(nodes []Node, context ActionContext)
}

type ShutDownWithBlockLagAction struct{}

func (action ShutDownWithBlockLagAction) execute(nodes []Node, context ActionContext) {
    log.Infof("Maximum last block height '%v' reported by %v.", context.MaximumBlockHeight, context.UpToDateNodes)
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

func shutDownNode(node Node) {
    _ = node.API.Shutdown()
    time.Sleep(time.Duration(200) * time.Millisecond)
    _ = node.API.Shutdown()
}

type PoolToolActionConfig struct {
    PoolID      string
    UserID      string
    GenesisHash string
}

type PostLastTipToPoolToolAction struct {
    Config PoolToolActionConfig
}

func (action PostLastTipToPoolToolAction) execute(nodes []Node, context ActionContext) {
    go pooltool.PostLatestTip(context.MaximumBlockHeight, action.Config.PoolID, action.Config.UserID, action.Config.GenesisHash)
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
                    fmt.Sprintf("Node '%v' has fallen behind %v blocks.", peer.Name, lag))
            }
        }
    }
}

type ShutDownWhenStuck struct{}

func (action ShutDownWhenStuck) execute(nodes []Node, context ActionContext) {
    if context.TimeSettings != nil {
        for p := range nodes {
            peer := nodes[p]
            lastBlock, found := context.LastBlockMap[peer.Name]
            if peer.MaxTimeSinceLastBlock <= 0 { // ignore nodes that have not set a max duration.
                continue
            }
            if found {
                mostRecentBlockDate := cardano.MakeFullSlotDate(lastBlock.LastBlockSlotDate, *context.TimeSettings)
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

func (action ReportStuckPerEmailAction) execute(nodes []Node, context ActionContext) {
    if context.TimeSettings != nil {
        for p := range nodes {
            peer := nodes[p]
            lastBlock, found := context.LastBlockMap[peer.Name]
            if peer.MaxTimeSinceLastBlock <= 0 { // ignore nodes that have not set a max duration.
                continue
            }
            if found {
                mostRecentBlockDate := cardano.MakeFullSlotDate(lastBlock.LastBlockSlotDate, *context.TimeSettings)
                diff := time.Now().Sub(mostRecentBlockDate.GetEndDateTime())
                if diff > peer.MaxTimeSinceLastBlock {
                    sendEmailReport(action.Config, fmt.Sprintf("[THOR][%v] Report Blockchain Stuck.", peer.Name), "")
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
