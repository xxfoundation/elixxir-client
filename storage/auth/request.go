package auth

import (
	"gitlab.com/elixxir/client/interfaces/contact"
	"sync"
)

type requestType uint

const (
	Sent    requestType = 0
	Receive requestType = 1
)

type request struct {
	rt requestType
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
