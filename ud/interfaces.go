package ud

import (
	"gitlab.com/elixxir/client/api"
	"gitlab.com/elixxir/client/single"
	"gitlab.com/elixxir/client/storage/user"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/primitives/id"
)

// CMix is a sub-interface of the cmix.Client. It contains the methods
// relevant to what is used in this package.
type CMix interface {
	// CMix is passed down into the single use package,
	// and thus has to adhere to the sub-interface defined in that package
	single.CMix
}

// E2E is a sub-interface of the e2e.Handler. It contains the methods
// relevant to what is used in this package.
type E2E interface {
	// GetGroup returns the cyclic group used for end to end encruption
	GetGroup() *cyclic.Group

	// GetReceptionID returns the default IDs
	GetReceptionID() *id.ID
}

// UserInfo is a sub-interface for the user.User object in storage.
// It contains the methods relevant to what is used in this package.
type UserInfo interface {
	PortableUserInfo() user.Info
	GetReceptionRegistrationValidationSignature() []byte
}

// NetworkStatus is an interface for the api.Client's
// NetworkFollowerStatus method.
type NetworkStatus func() api.Status
