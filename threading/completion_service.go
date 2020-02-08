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

// a completion service takes a function and a list of inputs for this
// function. It computes the function for all the given inputs in parallel and
// returns the result of the computation in a list. The order of the response
// list must not correspond with the list of inputs. The context field of the
// response can be used to connect response with a certain input.
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
