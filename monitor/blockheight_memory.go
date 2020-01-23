package monitor

import (
    "fmt"
    "github.com/sobitada/go-jormungandr/api"
    "github.com/sobitada/thor/utils"
    "math/big"
)

// block height memory of all nodes for judging their health.
type blockHeightMemory struct {
    n     int
    mem   map[string][]*big.Int
    nodes []string
}

// creates a new block height memory. n specifies the number of checkpoints
// that shall be remembered. This method expects a list of nodes for which
// this memory shall be constructed is expected.
func createBlockHeightMemory(nodes []string, n int) *blockHeightMemory {
    emptyList := make([]*big.Int, n)
    for i := 0; i < n; i++ {
        emptyList[i] = new(big.Int).SetInt64(-1)
    }
    mem := make(map[string][]*big.Int)
    for i := range nodes {
        mem[nodes[i]] = emptyList
    }
    return &blockHeightMemory{n: n, mem: mem, nodes: nodes}
}

// adds the block heights for all given nodes of a new checkpoint.
func (m *blockHeightMemory) addBlockHeights(blockMap map[string]api.NodeStatistic) {
    for i := range m.nodes {
        name := m.nodes[i]
        var entry *big.Int
        stat, found := blockMap[name]
        if found {
            entry = stat.LastBlockHeight
        } else {
            entry = new(big.Int).SetInt64(-1)
        }
        m.mem[name] = append([]*big.Int{entry}, m.mem[name][:m.n]...)
    }
}

// computes the difference of the block height to the maximum reported
// block height for each of the given nodes.
func (m *blockHeightMemory) getDiff() map[string][]*big.Int {
    diffMap := make(map[string][]*big.Int)
    for n := range m.nodes {
        diffMap[m.nodes[n]] = make([]*big.Int, m.n)
    }
    for i := 0; i < m.n; i++ {
        currentMap := make(map[string]*big.Int)
        for n := range m.nodes {
            currentMap[m.nodes[n]] = m.mem[m.nodes[n]][i]
        }
        maxHeight, _ := utils.MaxInt(currentMap)
        for name, height := range currentMap {
            diffMap[name][i] = new(big.Int).Sub(maxHeight, height)
        }
    }
    return diffMap
}

// computes the health of all the given nodes. it applies a reverse
// weighting by time, i.e. that the reported difference of past
// checkpoints have a lower weight than new checkpoints. The exact
// function is:
//
//                     diff_x * squrt((n-x)/n), where
//
//  x ........ is the x-th entry in the memory.
//  n ........ the total number of entries in memory.
//  diff_x ... the difference at checkpoint entry at position x.
//
func (m *blockHeightMemory) computeHealth() map[string]*big.Float {
    confMap := make(map[string]*big.Float)
    for name, history := range m.getDiff() {
        var conf = new(big.Float)
        for i := range history {
            diff_x := new(big.Float).SetInt(history[i])
            n := new(big.Float).SetInt64(int64(len(history)))
            x := new(big.Float).SetInt64(int64(i))
            weight_x := new(big.Float).Sqrt(new(big.Float).Quo(new(big.Float).Sub(n,x), n))
            h_x := new(big.Float).Mul(diff_x, weight_x)
            // accumulate health
            conf = new(big.Float).Add(conf, h_x)
        }
        confMap[name] = conf
    }
    return confMap
}

// string representation of the block height memory.
func (m *blockHeightMemory) String() string {
    var result string
    for name, num := range m.mem {
        result += fmt.Sprintf("%v=%v;", name, num)
    }
    return result + "\n"
}
