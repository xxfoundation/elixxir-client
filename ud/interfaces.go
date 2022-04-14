package ud

import (
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/interfaces/user"
	"gitlab.com/elixxir/client/single"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
)

type Userinfo interface {
	PortableUserInfo() user.Info
	GetUsername() (string, error)
	GetReceptionRegistrationValidationSignature() []byte
}

type SingleInterface interface {
	TransmitRequest(recipient contact.Contact, tag string, payload []byte,
		callback single.Response, param single.RequestParams, net cmix.Client, rng csprng.Source,
		e2eGrp *cyclic.Group) (id.Round, receptionID.EphemeralIdentity, error)
	StartProcesses() (stoppable.Stoppable, error)
}
