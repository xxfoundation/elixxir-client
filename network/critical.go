package network

import (
	"gitlab.com/elixxir/client/network/health"
)

type Manager struct {
	storage *CmixMessageBuffer
	trigger chan bool
	hm      *health.Monitor
}

func NewManager()
