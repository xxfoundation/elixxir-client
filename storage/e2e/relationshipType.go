package e2e

import (
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
)

type RelationshipType uint

const (
	Send RelationshipType = iota
	Receive
)

func (rt RelationshipType) String() string {
	switch rt {
	case Send:
		return "Send"
	case Receive:
		return "Receive"
	default:
		return fmt.Sprintf("Unknown relationship type: %d", rt)
	}
}

func (rt RelationshipType) prefix() string {
	switch rt {
	case Send:
		return "Send"
	case Receive:
		return "Receive"
	default:
		jww.FATAL.Panicf("No prefix for relationship type: %s", rt)
	}
	return ""
}
