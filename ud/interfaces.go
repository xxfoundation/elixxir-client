////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package ud

import (
	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/e2e"
	"gitlab.com/elixxir/client/v4/event"
	"gitlab.com/elixxir/client/v4/single"
	"gitlab.com/elixxir/client/v4/storage"
	"gitlab.com/elixxir/client/v4/xxdk"
	"gitlab.com/elixxir/crypto/fastRNG"
)

//////////////////////////////////////////////////////////////////////////////////////
// UD sub-interfaces
/////////////////////////////////////////////////////////////////////////////////////

// udCmix is a sub-interface of the cmix.Client. It contains the methods
// relevant to what is used in this package.
type udCmix interface {
	// Cmix within the single package is what udCmix must adhere to when passing
	// arguments through to methods in the single package.
	single.Cmix
}

// udE2e is a sub-interface of the xxdk.E2e. It contains the methods
// relevant to what is used in this package.
type udE2e interface {
	GetReceptionIdentity() xxdk.ReceptionIdentity
	GetCmix() cmix.Client
	GetE2E() e2e.Handler
	GetEventReporter() event.Reporter
	GetRng() *fastRNG.StreamGenerator
	GetStorage() storage.Session
	GetTransmissionIdentity() xxdk.TransmissionIdentity
	GetBackupContainer() *xxdk.Container
}

// udNetworkStatus is an interface for the xxdk.Cmix's
// NetworkFollowerStatus method.
type udNetworkStatus func() xxdk.Status
