package utils

import "math/big"

// scans the map of peers with their reported block height, and
// then returns the highest reported block height as well as a
// list of peers (more specifically their name) that reported
// exactly this height.
func MaxInt(blockHeightMap map[string]*big.Int) (*big.Int, []string) {
    maxV := new(big.Int).SetInt64(0)
    maxHeightPeersMap := make(map[string][]string)
    for key, value := range blockHeightMap {
        if value.Cmp(maxV) >= 0 {
            list, found := maxHeightPeersMap[value.String()]
            if found {
                maxHeightPeersMap[value.String()] = append(list, key)
            } else {
                maxHeightPeersMap[value.String()] = []string{key}
            }
            maxV = value
        }
    }
    peers, _ := maxHeightPeersMap[maxV.String()]
    return maxV, peers
}

// scans the map of peers with their reported healthiness value, and
// then returns the highest reported block height as well as a
// list of peers (more specifically their name) that reported
// exactly this height.
func MinFloat(floatMap map[string]*big.Float) (*big.Float, []string) {
    minV := new(big.Float).SetInf(false)
    minConfMap := make(map[string][]string)
    for key, value := range floatMap {
        if minV.Cmp(value) >= 0 {
            list, found := minConfMap[value.String()]
            if found {
                minConfMap[value.String()] = append(list, key)
            } else {
                minConfMap[value.String()] = []string{key}
            }
            minV = value
        }
    }
    peers, _ := minConfMap[minV.String()]
    return minV, peers
}