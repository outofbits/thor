package leader

import "fmt"

type invalidArgument struct {
    Method string
    Reason string
}

func (error invalidArgument) Error() string {
    return fmt.Sprintf("Invalid argument passed to %v. %v", error.Method, error.Reason)
}
