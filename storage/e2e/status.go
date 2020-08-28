package e2e

import "fmt"

type Status uint8

const (
	// Active sessions have keys remaining that can be used for messages
	Active Status = iota
	// RekeyNeeded sessions have keys remaining for messages, but should be rekeyed immediately
	RekeyNeeded
	// Empty sessions can't be used for more messages, but can be used for rekeys
	Empty
	// RekeyEmpty sessions are totally empty and no longer have enough keys left for a rekey, much less messages
	RekeyEmpty
)

func (a Status) String() string {
	switch a {
	case Active:
		return "Active"
	case RekeyNeeded:
		return "Rekey Needed"
	case Empty:
		return "Empty"
	case RekeyEmpty:
		return "Rekey Empty"
	default:
		return fmt.Sprintf("Unknown: %v", int(a))
	}
}
