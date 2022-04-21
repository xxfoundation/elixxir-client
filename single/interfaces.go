package single

import (
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"time"
)

// CMix is a sub-interface of the cmix.Client. It contains the methods
// relevant to what is used in this package.
type CMix interface {
	IsHealthy() bool
	GetAddressSpace() uint8
	GetMaxMessageLength() int
	DeleteClientFingerprints(identity *id.ID)
	AddFingerprint(identity *id.ID, fingerprint format.Fingerprint,
		mp message.Processor) error
	AddIdentity(id *id.ID, validUntil time.Time, persistent bool)
	Send(recipient *id.ID, fingerprint format.Fingerprint,
		service message.Service, payload, mac []byte, cmixParams cmix.CMIXParams) (
		id.Round, ephemeral.Id, error)
	AddService(clientID *id.ID, newService message.Service,
		response message.Processor)
	DeleteService(clientID *id.ID, toDelete message.Service,
		processor message.Processor)
	GetInstance() *network.Instance
	CheckInProgressMessages()
}
