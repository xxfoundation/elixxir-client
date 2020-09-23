package context

import (
	"gitlab.com/elixxir/client/context/message"
	"gitlab.com/elixxir/client/context/params"
	"gitlab.com/elixxir/client/context/stoppable"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
)

type NetworkManager interface {
	SendE2E(m message.Send, p params.E2E) ([]id.Round, error)
	SendUnsafe(m message.Send, p params.Unsafe) ([]id.Round, error)
	SendCMIX(message format.Message, p params.CMIX) (id.Round, error)
	GetInstance() *network.Instance
	GetHealthTracker() HealthTracker
	RegisterWithPermissioning(string) ([]byte, error)
	GetRemoteVersion() (string, error)
	GetStoppable() stoppable.Stoppable
}

type HealthTracker interface {
	AddChannel(chan bool)
	AddFunc(f func(bool))
	IsHealthy() bool
}
