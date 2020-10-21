package auth

import (
	"gitlab.com/elixxir/client/interfaces/contact"
	"sync"
)

type RequestType uint

const (
	Sent    RequestType = 0
	Receive RequestType = 1
)

type request struct {
	rt RequestType
	//data if sent
	sent *SentRequest
	//data if receive
	receive *contact.Contact
	//mux to ensure there isnt concurent access
	mux sync.Mutex
}

type requestDisk struct {
	T  uint
	ID []byte
}
