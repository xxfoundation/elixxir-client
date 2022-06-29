////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package connect

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/auth"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	clientE2e "gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/client/xxdk"
	"gitlab.com/elixxir/crypto/contact"
)

// clientAuthCallback provides callback functionality for interfacing between
// auth.State and Connection. This is used both for blocking creation of a
// Connection object until the auth Request is confirmed and for dynamically
// building new Connection objects when an auth Request is received.
type clientAuthCallback struct {
	// Used for signaling confirmation of E2E partnership
	confirmCallback Callback
	requestCallback Callback

	// Used for building new Connection objects
	connectionE2e    clientE2e.Handler
	connectionParams Params
	authState        auth.State
}

// getClientAuthCallback returns a callback interface to be passed into the creation
// of an auth.State object.
// it will accept requests only if a request callback is passed in
func getClientAuthCallback(confirm, request Callback, e2e clientE2e.Handler,
	auth auth.State, params Params) *clientAuthCallback {
	return &clientAuthCallback{
		confirmCallback:  confirm,
		requestCallback:  request,
		connectionE2e:    e2e,
		connectionParams: params,
		authState:        auth,
	}
}

// Confirm will be called when an auth Confirm message is processed.
func (a clientAuthCallback) Confirm(requestor contact.Contact,
	_ receptionID.EphemeralIdentity, _ rounds.Round) {
	jww.DEBUG.Printf("Connection auth request for %s confirmed",
		requestor.ID.String())
	defer a.authState.DeletePartnerCallback(requestor.ID)

	// After confirmation, get the new partner
	newPartner, err := a.connectionE2e.GetPartner(requestor.ID)
	if err != nil {
		jww.ERROR.Printf("Unable to build connection with "+
			"partner %s: %+v", requestor.ID, err)
		// Send a nil connection to avoid hold-ups down the line
		if a.confirmCallback != nil {
			a.confirmCallback(nil)
		}
		return
	}

	// Return the new Connection object
	if a.confirmCallback != nil {
		a.confirmCallback(BuildConnection(newPartner, a.connectionE2e,
			a.authState, a.connectionParams))
	}
}

// Request will be called when an auth Request message is processed.
func (a clientAuthCallback) Request(contact.Contact,
	receptionID.EphemeralIdentity, rounds.Round) {
}

// Reset will be called when an auth Reset operation occurs.
func (a clientAuthCallback) Reset(contact.Contact,
	receptionID.EphemeralIdentity, rounds.Round) {
}

// serverAuthCallback provides callback functionality for interfacing between
// auth.State and Connection. This is used both for blocking creation of a
// Connection object until the auth Request is confirmed and for dynamically
// building new Connection objects when an auth Request is received.
type serverAuthCallback struct {
	// Used for signaling confirmation of E2E partnership
	confirmCallback Callback
	requestCallback Callback

	// Used for building new Connection objects
	connectionParams Params
}

// getServerAuthCallback returns a callback interface to be passed into the creation
// of a xxdk.E2e object.
// it will accept requests only if a request callback is passed in
func getServerAuthCallback(confirm, request Callback, params Params) *serverAuthCallback {
	return &serverAuthCallback{
		confirmCallback:  confirm,
		requestCallback:  request,
		connectionParams: params,
	}
}

// Confirm will be called when an auth Confirm message is processed.
func (a serverAuthCallback) Confirm(contact.Contact,
	receptionID.EphemeralIdentity, rounds.Round, *xxdk.E2e) {
}

// Request will be called when an auth Request message is processed.
func (a serverAuthCallback) Request(requestor contact.Contact,
	_ receptionID.EphemeralIdentity, _ rounds.Round, e2e *xxdk.E2e) {
	if a.requestCallback == nil {
		jww.ERROR.Printf("Received a request when requests are" +
			"not enable, will not accept")
	}
	_, err := e2e.GetAuth().Confirm(requestor)
	if err != nil {
		jww.ERROR.Printf("Unable to build connection with "+
			"partner %s: %+v", requestor.ID, err)
		// Send a nil connection to avoid hold-ups down the line
		a.requestCallback(nil)
	}
	// After confirmation, get the new partner
	newPartner, err := e2e.GetE2E().GetPartner(requestor.ID)
	if err != nil {
		jww.ERROR.Printf("Unable to build connection with "+
			"partner %s: %+v", requestor.ID, err)
		// Send a nil connection to avoid hold-ups down the line
		a.requestCallback(nil)

		return
	}

	// Return the new Connection object
	a.requestCallback(BuildConnection(newPartner, e2e.GetE2E(),
		e2e.GetAuth(), a.connectionParams))
}

// Reset will be called when an auth Reset operation occurs.
func (a serverAuthCallback) Reset(contact.Contact,
	receptionID.EphemeralIdentity, rounds.Round, *xxdk.E2e) {
}
