package context

import (
	"gitlab.com/elixxir/client/context/message"
	"gitlab.com/elixxir/client/context/params"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
)

type NetworkManager interface {
	SendE2E(m message.Send, e2eP params.E2E, cmixP params.CMIX) ([]id.Round, error)
	SendUnsafe(m message.Send) ([]id.Round, error)
	SendCMIX(message format.Message) (id.Round, error)
	GetInstance() *network.Instance
	GetHealthTracker() HealthTracker
}

type HealthTracker interface {
	AddChannel(chan bool)
	AddFunc(f func(bool))
	IsHealthy() bool
}
