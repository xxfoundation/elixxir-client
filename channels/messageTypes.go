package channels

import "fmt"

type MessageType uint32

const (
	Text      = MessageType(1)
	AdminText = MessageType(2)
	Reaction  = MessageType(3)
)

func (mt MessageType) String() string {
	switch mt {
	case Text:
		return "Text"
	case AdminText:
		return "AdminText"
	case Reaction:
		return "Reaction"
	default:
		return fmt.Sprintf("Unknown messageType %d", mt)
	}
}
