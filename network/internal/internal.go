package internal

import (
	"gitlab.com/elixxir/client/context"
	"gitlab.com/elixxir/client/network/health"
	"gitlab.com/elixxir/comms/client"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/xx_network/primitives/id"
)

type Internal struct {
	*context.Context

	// Comms pointer to send/recv messages
	Comms *client.Comms
	//contains the health tracker which keeps track of if from the client's
	//perspective, the network is in good condition
	Health *health.Tracker
	//ID of the node
	Uid *id.ID
	//contains the network instance
	Instance *network.Instance

	//channels
	NodeRegistration chan network.NodeGateway
	//local pointer to user ID because it is used often

}
