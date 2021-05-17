package interfaces

// this interface is used to allow the follower to to be stopped later if it
// fails

type Running interface {
	IsRunning() bool
}
