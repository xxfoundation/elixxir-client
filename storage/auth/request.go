package auth

import (
	"gitlab.com/elixxir/client/interfaces/contact"
	"sync"
)

type RequestType uint

const (
	Sent    RequestType = 1
	Receive RequestType = 2
)

type request struct {
	rt RequestType

	// Data if sent
	sent *SentRequest

	// Data if receive
	receive *contact.Contact

	// mux to ensure there is not concurrent access
	mux sync.Mutex
}

type requestDisk struct {
	T  uint
	ID []byte
}
