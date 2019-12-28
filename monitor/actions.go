package monitor

import (
    log "github.com/sirupsen/logrus"
    "github.com/sobitada/thor/pooltool"
    "time"
)

type ActionContext struct {
    BlockHeightMap map[string]uint32
}

type Action interface {
    execute(nodes []Node, context ActionContext)
}

type ShutDownWithBlockLagAction struct {
}

func (action ShutDownWithBlockLagAction) execute(nodes []Node, context ActionContext) {
    height, pools := max(context.BlockHeightMap)
    log.Infof("Maximum last block height '%v' reported by %v.", height, pools)
    for p := range nodes {
        peer := nodes[p]
        peerBlockHeight, found := context.BlockHeightMap[peer.Name]
        if found {
            if peerBlockHeight < (height - peer.MaxBlockLag) {
                log.Warnf("[%s] Pool has fallen behind %v blocks.", peer.Name, height-peerBlockHeight)
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

type PostLastTipToPoolToolAction struct {
    PoolID      string
    UserID      string
    GenesisHash string
}

func (action PostLastTipToPoolToolAction) execute(nodes []Node, context ActionContext) {
    height, _ := max(context.BlockHeightMap)
    go pooltool.PostLatestTip(height, action.PoolID, action.UserID, action.GenesisHash)
}
