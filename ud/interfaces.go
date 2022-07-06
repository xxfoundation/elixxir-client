package ud

import (
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/client/event"
	"gitlab.com/elixxir/client/single"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/xxdk"
	"gitlab.com/elixxir/crypto/fastRNG"
)

// CMix is a sub-interface of the cmix.Client. It contains the methods
// relevant to what is used in this package.
type CMix interface {
	// CMix is passed down into the single use package,
	// and thus has to adhere to the sub-interface defined in that package
	single.Cmix
}

// E2E is a sub-interface of the xxdk.E2e. It contains the methods
// relevant to what is used in this package.
type E2E interface {
	GetReceptionIdentity() xxdk.ReceptionIdentity
	GetCmix() cmix.Client
	GetE2E() e2e.Handler
	GetEventReporter() event.Reporter
	GetRng() *fastRNG.StreamGenerator
	GetStorage() storage.Session
	GetTransmissionIdentity() xxdk.TransmissionIdentity
}

// NetworkStatus is an interface for the xxdk.Cmix's
// NetworkFollowerStatus method.
type NetworkStatus func() xxdk.Status
