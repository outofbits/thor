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

// max duration returns the larger of x or y.
func MaxDuration(x, y time.Duration) time.Duration {
    if x < y {
        return y
    }
    return x
}

// min duration returns the smaller of x or y.
func MinDuration(x, y time.Duration) time.Duration {
    if x > y {
        return y
    }
    return x
}
