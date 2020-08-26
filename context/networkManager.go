package context

import (
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
)

type NetworkManager interface {
	SendE2E(m Message) ([]id.Round, error)
	SendUnsafe(m Message) ([]id.Round, error)
	SendCMIX(message format.Message) (id.Round, error)
	GetRekeyChan() chan id.ID
	GetInstance() *network.Instance
	//placeholder to stop active threads
	Kill() bool
}
