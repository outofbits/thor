package monitor

import (
    "fmt"
    log "github.com/sirupsen/logrus"
    "github.com/sobitada/go-jormungandr/api/dto"
    "github.com/sobitada/thor/pooltool"
    "net/smtp"
    "strings"
    "time"
)

type ActionContext struct {
    BlockHeightMap     map[string]uint32
    MaximumBlockHeight uint32
    UpToDateNodes      []string
    LastBlockMap       map[string]dto.NodeStatistic
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
            if peerBlockHeight < (context.MaximumBlockHeight - peer.MaxBlockLag) {
                log.Warnf("[%s] Pool has fallen behind %v blocks.", peer.Name, context.MaximumBlockHeight-peerBlockHeight)
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
            if peerBlockHeight < (context.MaximumBlockHeight - peer.MaxBlockLag) {
                lag := context.MaximumBlockHeight - peerBlockHeight
                go sendEmailReport(action.Config, fmt.Sprintf("[THOR][%v] Report of Block Lag.", peer.Name),
                    fmt.Sprintf("Node '%v' has fallen behind %v blocks.", peer.Name, lag))
            }
        }
    }
}

type ReportStuckPerEmailAction struct {
    Config EmailActionConfig
}

func (action ReportStuckPerEmailAction) execute(nodes []Node, context ActionContext) {
    for p := range nodes {
        peer := nodes[p]
        lastBlock, found := context.LastBlockMap[peer.Name]
        if peer.MaxTimeSinceLastBlock <= 0 { // ignore nodes that have not set a max duration.
            continue
        }
        if found {
            lastBlockTime, err := time.Parse(time.RFC3339, lastBlock.LastBlockTime)
            if err == nil {
                diff := time.Now().Sub(lastBlockTime)
                if diff > peer.MaxTimeSinceLastBlock {
                    sendEmailReport(action.Config, fmt.Sprintf("[THOR][%v] Report Blockchain Stuck.", peer.Name), "")
                }
            } else {
                log.Warn("Could not parse the given timestamp %v. %v", lastBlock.LastBlockTime, err.Error())
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
