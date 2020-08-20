package key

import "fmt"

type Status uint8

const (
	Active Status = iota
	RekeyNeeded
	Empty
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
