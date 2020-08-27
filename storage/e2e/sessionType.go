package e2e

type SessionType uint8

const (
	Send SessionType = iota
	Receive
)

func (st SessionType) String() string {
	switch st {
	case Send:
		return "Send"
	case Receive:
		return "Receive"
	default:
		return "Unknown"
	}
}
