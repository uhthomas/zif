package jobs

// Takes in data, returns a stream of data
type Job func(<-chan interface{}, ...interface{}) chan<- interface{}
