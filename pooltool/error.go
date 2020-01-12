package pooltool

import "fmt"

type poolToolAPIException struct {
    URL        string
    StatusCode int
    Reason     string
}

func (e poolToolAPIException) Error() string {
    return fmt.Sprintf("Pool Tool API method '%v' failed with status code %v. %v", e.URL, e.StatusCode, e.Reason)
}
