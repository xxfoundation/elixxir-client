package params

import "fmt"

type E2E struct {
	Type SendType
}

func GetDefaultE2E() E2E {
	return E2E{Type: Standard}
}

type SendType uint8

const (
	Standard    SendType = 0
	KeyExchange SendType = 1
)

func (st SendType) String() string {
	switch st {
	case Standard:
		return "Standard"
	case KeyExchange:
		return "KeyExchange"
	default:
		return fmt.Sprintf("Unknown SendType %v", uint8(st))
	}
}
