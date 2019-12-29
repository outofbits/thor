package monitor

import (
    "fmt"
    log "github.com/sirupsen/logrus"
    "github.com/sobitada/thor/pooltool"
    "net/smtp"
    "net/url"
    "strings"
    "time"
)

type ActionContext struct {
    BlockHeightMap     map[string]uint32
    MaximumBlockHeight uint32
    UpToDateNodes      []string
}

type Action interface {
    execute(nodes []Node, context ActionContext)
}

type ShutDownWithBlockLagAction struct{}

func (action ShutDownWithBlockLagAction) execute(nodes []Node, context ActionContext) {
    log.Infof("Maximum last block height '%v' reported by %v.", context.MaximumBlockHeight, context.UpToDateNodes)
    for p := range nodes {
        peer := nodes[p]
        peerBlockHeight, found := context.BlockHeightMap[peer.Name]
        if found {
            if peerBlockHeight < (context.MaximumBlockHeight - peer.MaxBlockLag) {
                log.Warnf("[%s] Pool has fallen behind %v blocks.", peer.Name, context.MaximumBlockHeight-peerBlockHeight)
                go shutDownNode(peer)
            }
        }
    }
}

// shuts down the peer
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
    ServerURL            url.URL
    Authentication       smtp.Auth
}

type ReportBlockLagPerEmailAction struct {
    Config EmailActionConfig
}

func (action ReportBlockLagPerEmailAction) execute(nodes []Node, context ActionContext) {
    for p := range nodes {
        peer := nodes[p]
        peerBlockHeight, found := context.BlockHeightMap[peer.Name]
        if found {
            if peerBlockHeight < (context.MaximumBlockHeight - peer.MaxBlockLag) {
                lag := context.MaximumBlockHeight - peerBlockHeight
                go sendEmailReport(action.Config, fmt.Sprintf("[THOR][%v] Report of Block Lag.", peer.Name),
                    fmt.Sprintf("Node '%v' has fallen behind %v blocks.", peer.Name, lag))
            }
        }
    }
}

func sendEmailReport(config EmailActionConfig, subject string, message string) {
    msg := []byte(fmt.Sprintf("To: %v\r\nSubject: %v\r\n%v\r\n", strings.Join(config.DestinationAddresses, ";"), subject, message))
    err := smtp.SendMail(config.ServerURL.String(), config.Authentication, config.SourceAddress, config.DestinationAddresses, msg)
    if err != nil {
        log.Errorf("Could not send email. %v", err.Error())
    }
}
