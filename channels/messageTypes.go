package channels

import "fmt"

type MessageType uint32

const (
	Text      = MessageType(1)
	AdminText = MessageType(2)
)

func (mt MessageType) String() string {
	switch mt {
	case Text:
		return "Text"
	default:
		return fmt.Sprintf("Unknown messageType %d", mt)
	}
}
