///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package internal

import (
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/network/health"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/switchboard"
	"gitlab.com/elixxir/comms/client"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/xx_network/primitives/id"
)

type Internal struct {
	Session     *storage.Session
	Switchboard *switchboard.Switchboard
	//generic RNG for client

	// Comms pointer to send/recv messages
	Comms *client.Comms
	//contains the health tracker which keeps track of if from the client's
	//perspective, the network is in good condition
	Health *health.Tracker
	//ID which messages are sent as
	TransmissionID *id.ID
	//ID which messages are received as
	ReceptionID *id.ID
	//contains the network instance
	Instance *network.Instance

	//channels
	NodeRegistration chan network.NodeGateway

	// Event Reporting
	Events interfaces.EventManager
}
