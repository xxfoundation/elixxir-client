package stoppable

import "time"

// Interface for stopping a goroutine.
type Stoppable interface {
	Close(timeout time.Duration) error
	IsRunning() bool
	Name() string
}
