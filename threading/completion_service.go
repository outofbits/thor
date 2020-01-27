package threading

// a response of a performed function with
// context, data, and potentially with
// error information.
type Response struct {
    Context interface{}
    Data    interface{}
    Error   error
}

func perform(input interface{}, channel chan Response, function func(input interface{}) Response) {
    channel <- function(input)
}

func Complete(inputs []interface{}, function func(input interface{}) Response) []Response {
    response := make([]Response, len(inputs))
    channel := make(chan Response)
    // perform the requests
    for i := 0; i < len(inputs); i++ {
        go perform(inputs[i], channel, function)
    }
    // get the responses.
    for i := 0; i < len(inputs); i++ {
        response[i] = <-channel
    }
    return response
}
