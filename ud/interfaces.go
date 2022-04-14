package ud

import (
	"gitlab.com/elixxir/client/api"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/single"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage/user"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
)

// UserInfo is an interface for the user.User object.
type UserInfo interface {
	PortableUserInfo() user.Info
	GetUsername() (string, error)
	GetReceptionRegistrationValidationSignature() []byte
}

// NetworkStatus is an interface for the api.Client's
// NetworkFollowerStatus method.
type NetworkStatus func() api.Status

// todo: this may not be needed. if so, remove.
type SingleInterface interface {
	TransmitRequest(recipient contact.Contact, tag string, payload []byte,
		callback single.Response, param single.RequestParams, net cmix.Client, rng csprng.Source,
		e2eGrp *cyclic.Group) (id.Round, receptionID.EphemeralIdentity, error)
	StartProcesses() (stoppable.Stoppable, error)
}
