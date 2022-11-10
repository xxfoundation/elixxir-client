////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package connect

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v5/auth"
	"gitlab.com/elixxir/client/v5/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v5/cmix/rounds"
	clientE2e "gitlab.com/elixxir/client/v5/e2e"
	"gitlab.com/elixxir/client/v5/xxdk"
	"gitlab.com/elixxir/crypto/contact"
)

// clientAuthCallback provides callback functionality for interfacing between
// auth.State and Connection. This is used for building new Connection
// objects when an auth Confirm is received.
type clientAuthCallback struct {
	// Used for signaling confirmation of E2E partnership
	confirmCallback Callback

	// Used for building new Connection objects
	connectionE2e    clientE2e.Handler
	connectionParams xxdk.E2EParams
	authState        auth.State
}

// getClientAuthCallback returns an auth.Callbacks interface to be passed into the creation
// of an auth.State object for connect clients.
func getClientAuthCallback(confirm Callback, e2e clientE2e.Handler,
	auth auth.State, params xxdk.E2EParams) *clientAuthCallback {
	return &clientAuthCallback{
		confirmCallback:  confirm,
		connectionE2e:    e2e,
		connectionParams: params,
		authState:        auth,
	}
}

// Confirm will be called when an auth Confirm message is processed.
func (a clientAuthCallback) Confirm(requestor contact.Contact,
	_ receptionID.EphemeralIdentity, _ rounds.Round) {
	jww.DEBUG.Printf("Connection auth confirm for %s received",
		requestor.ID.String())
	defer a.authState.DeletePartnerCallback(requestor.ID)

	// After confirmation, get the new partner
	newPartner, err := a.connectionE2e.GetPartner(requestor.ID)
	if err != nil {
		jww.ERROR.Printf("Unable to build connection with "+
			"partner %s: %+v", requestor.ID, err)
		// Send a nil connection to avoid hold-ups down the line
		a.confirmCallback(nil)
		return
	}

	// Return the new Connection object
	a.confirmCallback(BuildConnection(newPartner, a.connectionE2e,
		a.authState, a.connectionParams))
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
// auth.State and Connection. This is used for building new Connection
//objects when an auth Request is received.
type serverAuthCallback struct {
	// Used for signaling confirmation of E2E partnership
	requestCallback Callback

	// Used to track stale connections
	cl *ConnectionList

	// Used for building new Connection objects
	connectionParams xxdk.E2EParams
}

// getServerAuthCallback returns an auth.Callbacks interface to be passed into the creation
// of a xxdk.E2e object for connect servers.
func getServerAuthCallback(request Callback, cl *ConnectionList,
	params xxdk.E2EParams) *serverAuthCallback {
	return &serverAuthCallback{
		requestCallback:  request,
		cl:               cl,
		connectionParams: params,
	}
}

// Confirm will be called when an auth Confirm message is processed.
func (a serverAuthCallback) Confirm(contact.Contact,
	receptionID.EphemeralIdentity, rounds.Round, *xxdk.E2e) {
}

// Request will be called when an auth Request message is processed.
func (a serverAuthCallback) Request(requestor contact.Contact,
	_ receptionID.EphemeralIdentity, _ rounds.Round, user *xxdk.E2e) {
	jww.DEBUG.Printf("Connection auth request for %s received",
		requestor.ID.String())

	// Auto-confirm the auth request
	_, err := user.GetAuth().Confirm(requestor)
	if err != nil {
		jww.ERROR.Printf("Unable to build connection with "+
			"partner %s: %+v", requestor.ID, err)
		return
	}

	// After confirmation, get the new partner
	newPartner, err := user.GetE2E().GetPartner(requestor.ID)
	if err != nil {
		jww.ERROR.Printf("Unable to build connection with "+
			"partner %s: %+v", requestor.ID, err)
		return
	}

	// Return the new Connection object
	c := BuildConnection(
		newPartner, user.GetE2E(), user.GetAuth(), a.connectionParams)
	a.cl.Add(c)
	a.requestCallback(c)
}

// Reset will be called when an auth Reset operation occurs.
func (a serverAuthCallback) Reset(requestor contact.Contact,
	receptionId receptionID.EphemeralIdentity, round rounds.Round, user *xxdk.E2e) {
	a.Request(requestor, receptionId, round, user)
}
