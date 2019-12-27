package pooltool

import (
    "fmt"
    "net/http"
    "net/url"
)

const poolToolTipURL string = "https://tamoq3vkbl.execute-api.us-west-2.amazonaws.com/prod/sharemytip"

// posts the given block height to the pool tool API using
// the given pool tool configuration, which specifies the user
// id, pool id and the genesis of the block chain for which the
// tip shall be registered.
func PostLatestTip(tip uint32, poolID string, userID string, genesisHash string) error {
    u, err := url.Parse(poolToolTipURL)
    if err == nil {
        q := u.Query()
        q.Set("poolid", poolID)
        q.Set("userid", userID)
        q.Set("genesispref", genesisHash)
        q.Set("mytip", fmt.Sprint(tip))
        u.RawQuery = q.Encode()
        response, err := http.Get(u.String())
        if err == nil {
            if response.StatusCode == 200 {
                return nil
            } else {
                print("ERROR, Code:", response.StatusCode, u.String())
            }
        }
        return err
    }
    return err
}
