////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package single

import (
	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v4/cmix/message"
	cMixMsg "gitlab.com/elixxir/client/v4/cmix/message"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"time"
)

// Receiver contains the callback interface for any handler which
// will process the reception of a single request. Used in Listen.
type Receiver interface {
	Callback(*Request, receptionID.EphemeralIdentity, []rounds.Round)
}

// Response contains the callback interface for any handler which
// will process the response of a single request. Used in TransmitRequest.
type Response interface {
	Callback(payload []byte, receptionID receptionID.EphemeralIdentity,
		rounds []rounds.Round, err error)
}

type Listener interface {
	// Stop unregisters the listener
	Stop()
}

// RequestCmix interface matches a subset of the cmix.Client methods used by the
// Request for easier testing.
type RequestCmix interface {
	GetMaxMessageLength() int
	Send(recipient *id.ID, fingerprint format.Fingerprint,
		service cMixMsg.Service, payload, mac []byte,
		cmixParams cmix.CMIXParams) (rounds.Round, ephemeral.Id, error)
	GetInstance() *network.Instance
}

// ListenCmix interface matches a subset of cmix.Client methods used for Listen.
type ListenCmix interface {
	RequestCmix
	AddFingerprint(identity *id.ID, fingerprint format.Fingerprint,
		mp cMixMsg.Processor) error
	AddService(
		clientID *id.ID, newService cMixMsg.Service, response cMixMsg.Processor)
	DeleteService(
		clientID *id.ID, toDelete cMixMsg.Service, processor cMixMsg.Processor)
	CheckInProgressMessages()
}

// Cmix is a sub-interface of the cmix.Client. It contains the methods relevant
// to what is used in this package.
type Cmix interface {
	IsHealthy() bool
	GetAddressSpace() uint8
	GetMaxMessageLength() int
	DeleteClientFingerprints(identity *id.ID)
	AddFingerprint(identity *id.ID, fingerprint format.Fingerprint,
		mp message.Processor) error
	AddIdentity(id *id.ID, validUntil time.Time, persistent bool)
	Send(recipient *id.ID, fingerprint format.Fingerprint,
		service message.Service, payload, mac []byte, cmixParams cmix.CMIXParams) (
		rounds.Round, ephemeral.Id, error)
	AddService(clientID *id.ID, newService message.Service,
		response message.Processor)
	DeleteService(clientID *id.ID, toDelete message.Service,
		processor message.Processor)
	GetInstance() *network.Instance
	CheckInProgressMessages()
}
