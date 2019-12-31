package monitor

import (
    "github.com/hako/durafmt"
    "math/big"
    "time"
)

// scans the map of peers with their reported block height, and
// then returns the highest reported block height as well as a
// list of peers (more specifically their name) that reported
// exactly this height.
func max(blockHeightMap map[string]*big.Int) (*big.Int, []string) {
    maxV := new(big.Int).SetInt64(0)
    maxHeightPeersMap := make(map[string][]string)
    for key, value := range blockHeightMap {
        if value.Cmp(maxV) > 0 {
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

// transforms the given up time into a human readable string.
func getHumanReadableUpTime(upTime time.Duration) string {
    fmt := durafmt.Parse(upTime)
    if fmt != nil {
        return fmt.String()
    } else {
        return "NA"
    }
}
