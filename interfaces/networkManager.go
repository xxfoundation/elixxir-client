package interfaces

import (
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/stoppable"
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
	GetStoppable() stoppable.Stoppable
}

//for use in key exchange which needs to be callable inside of network
type SendE2E func(m message.Send, p params.E2E) ([]id.Round, error)