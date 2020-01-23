package utils

import (
    "github.com/hako/durafmt"
    "time"
)

// transforms the given up time into a human readable string.
func GetHumanReadableUpTime(upTime time.Duration) string {
    fmt := durafmt.Parse(upTime)
    if fmt != nil {
        return fmt.String()
    } else {
        return "NA"
    }
}