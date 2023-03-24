////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package xxdk

import (
	"encoding/json"
	"gitlab.com/elixxir/client/v4/storage/utility"
	"time"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/auth"
	"gitlab.com/elixxir/client/v4/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/client/v4/e2e"
	"gitlab.com/elixxir/client/v4/e2e/rekey"
	"gitlab.com/elixxir/client/v4/storage/user"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/xx"
	"gitlab.com/xx_network/primitives/id"
)

// E2e object bundles a ReceptionIdentity with a Cmix object and can be used for
// high-level operations, such as connections.
type E2e struct {
	*Cmix
	auth        auth.State
	e2e         e2e.Handler
	backup      *Container
	e2eIdentity ReceptionIdentity
}

// AuthCallbacks is an adapter for the auth.Callbacks interface that allows for
// initializing an E2e object without an E2e-dependant auth.Callbacks.
type AuthCallbacks interface {
	Request(partner contact.Contact, receptionID receptionID.EphemeralIdentity,
		round rounds.Round, user *E2e)
	Confirm(partner contact.Contact, receptionID receptionID.EphemeralIdentity,
		round rounds.Round, user *E2e)
	Reset(partner contact.Contact, receptionID receptionID.EphemeralIdentity,
		round rounds.Round, user *E2e)
}

// Login creates a new E2e backed by the xxdk.Cmix persistent versioned.KV. It
// bundles a Cmix object with a ReceptionIdentity object and initializes the
// auth.State and e2e.Handler objects.
func Login(net *Cmix, callbacks AuthCallbacks,
	identity ReceptionIdentity, params E2EParams) (m *E2e, err error) {

	// If the given identity matches the stored ReceptionID,
	// then we are using a legacy ReceptionIdentity
	defaultReceptionId := net.GetStorage().PortableUserInfo().ReceptionID
	if identity.ID.Cmp(defaultReceptionId) {
		return loginLegacy(net, callbacks, identity, params)
	}

	// Otherwise, we are using a modern ReceptionIdentity
	return login(net, callbacks, identity, net.GetStorage().GetKV(), params)
}

// LoginEphemeral creates a new E2e backed by a totally ephemeral versioned.KV.
func LoginEphemeral(net *Cmix, callbacks AuthCallbacks,
	identity ReceptionIdentity, params E2EParams) (m *E2e, err error) {
	return login(net, callbacks, identity,
		&utility.KV{Local: versioned.NewKV(ekv.MakeMemstore())}, params)
}

// loginLegacy creates a new E2e backed by the xxdk.Cmix persistent
// versioned.KV. It uses the pre-generated transmission ID used by xxdk.Cmix.
// This function is designed to maintain backwards compatibility with previous
// xx messenger designs and should not be used for other purposes.
func loginLegacy(net *Cmix, callbacks AuthCallbacks,
	identity ReceptionIdentity, params E2EParams) (
	m *E2e, err error) {
	m = &E2e{
		Cmix:   net,
		backup: &Container{},
	}

	m.e2e, err = loadOrInitE2eLegacy(identity, net)
	if err != nil {
		return nil, err
	}
	net.GetCmix().AddIdentity(identity.ID, time.Time{}, true, nil)

	err = net.AddService(m.e2e.StartProcesses)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to add the e2e processes")
	}

	m.auth, err = auth.NewStateLegacy(net.GetStorage().GetKV(),
		net.GetCmix(), m.e2e, net.GetRng(), net.GetEventReporter(),
		params.Auth, params.Session,
		MakeAuthCallbacksAdapter(callbacks, m),
		m.backup.TriggerBackup)
	if err != nil {
		return nil, err
	}

	rsaKey, err := identity.GetRSAPrivateKey()
	if err != nil {
		return nil, err
	}
	m.e2eIdentity, err = buildReceptionIdentity(identity.ID, identity.Salt,
		rsaKey, m.e2e.GetGroup(), m.e2e.GetHistoricalDHPrivkey())
	return m, err
}

// login creates a new xxdk.E2e backed by the given versioned.KV.
func login(net *Cmix, callbacks AuthCallbacks, identity ReceptionIdentity,
	kv *utility.KV, params E2EParams) (m *E2e, err error) {

	// Verify the passed-in ReceptionIdentity matches its properties
	privatePem, err := identity.GetRSAPrivateKey()
	if err != nil {
		return nil, err
	}
	generatedId, err := xx.NewID(privatePem.Public(), identity.Salt, id.User)
	if err != nil {
		return nil, err
	}
	if !generatedId.Cmp(identity.ID) {
		return nil, errors.Errorf(
			"Given identity %s is invalid, generated ID does not match",
			identity.ID.String())
	}

	m = &E2e{
		Cmix:        net,
		backup:      &Container{},
		e2eIdentity: identity,
	}
	dhPrivKey, err := identity.GetDHKeyPrivate()
	if err != nil {
		return nil, err
	}

	// Load or init the new e2e storage
	e2eGrp := net.GetStorage().GetE2EGroup()
	m.e2e, err = e2e.Load(kv, net.GetCmix(), identity.ID, e2eGrp, net.GetRng(),
		net.GetEventReporter())
	if err != nil {
		// Initialize the e2e storage
		err = e2e.Init(kv, identity.ID, dhPrivKey, e2eGrp, params.Rekey)
		if err != nil {
			return nil, err
		}

		// Load the new e2e storage
		m.e2e, err = e2e.Load(kv, net.GetCmix(), identity.ID, e2eGrp,
			net.GetRng(), net.GetEventReporter())
		if err != nil {
			return nil, errors.WithMessage(
				err, "Failed to load a newly created e2e store")
		}
	}

	err = net.AddService(m.e2e.StartProcesses)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to add the e2e processes")
	}

	m.auth, err = auth.NewState(kv, net.GetCmix(), m.e2e, net.GetRng(),
		net.GetEventReporter(), params.Auth, params.Session,
		MakeAuthCallbacksAdapter(callbacks, m), m.backup.TriggerBackup)
	if err != nil {
		return nil, err
	}

	net.network.AddIdentity(identity.ID, time.Time{}, true, nil)
	jww.INFO.Printf("Client logged in: \n\tReceptionID: %s", identity.ID)
	return m, err
}

// loadOrInitE2eLegacy loads the e2e.Handler or makes a new one, generating a
// new E2E private key. It attempts to load via a legacy construction, then
// tries to load the modern one, creating a new modern ID if neither can be
// found.
func loadOrInitE2eLegacy(identity ReceptionIdentity, net *Cmix) (e2e.Handler, error) {
	e2eGrp := net.GetStorage().GetE2EGroup()
	kv := net.GetStorage().GetKV()

	// Try to load a legacy e2e handler
	e2eHandler, err := e2e.LoadLegacy(kv,
		net.GetCmix(), identity.ID, e2eGrp, net.GetRng(),
		net.GetEventReporter(), rekey.GetDefaultParams())
	if err != nil {
		jww.DEBUG.Printf("e2e.LoadLegacy error: %v", err)
		// If no legacy e2e handler exists, try to load a new one
		e2eHandler, err = e2e.Load(kv,
			net.GetCmix(), identity.ID, e2eGrp, net.GetRng(),
			net.GetEventReporter())
		if err != nil {
			jww.WARN.Printf("Failed to load e2e instance for %s, "+
				"creating a new one: %v", identity.ID, err)

			// Initialize the e2e storage
			privKey, err := identity.GetDHKeyPrivate()
			if err != nil {
				return nil, err
			}
			err = e2e.Init(kv, identity.ID, privKey, e2eGrp,
				rekey.GetDefaultParams())
			if err != nil {
				return nil, err
			}

			// Load the new e2e storage
			e2eHandler, err = e2e.Load(kv,
				net.GetCmix(), identity.ID, e2eGrp, net.GetRng(),
				net.GetEventReporter())
			if err != nil {
				return nil, errors.WithMessage(err,
					"Failed to load a newly created e2e store")
			}
		} else {
			jww.INFO.Printf("Loaded a modern e2e instance for %s", identity.ID)
		}
	} else {
		jww.INFO.Printf("Loaded a legacy e2e instance for %s", identity.ID)
	}

	return e2eHandler, nil
}

// GetReceptionIdentity returns a safe copy of the E2e ReceptionIdentity.
func (m *E2e) GetReceptionIdentity() ReceptionIdentity {
	return m.e2eIdentity.DeepCopy()
}

// ConstructProtoUserFile is a helper function that is used for proto client
// testing. This is used for development testing.
func (m *E2e) ConstructProtoUserFile() ([]byte, error) {

	// load the registration code
	regCode, err := m.GetStorage().GetRegCode()
	if err != nil {
		return nil, errors.WithMessage(err,
			"failed to register with permissioning")
	}

	transIdentity := m.Cmix.GetTransmissionIdentity()
	receptionIdentity := m.GetReceptionIdentity()
	privateKey, err := receptionIdentity.GetRSAPrivateKey()
	if err != nil {
		return nil, err
	}

	Usr := user.Proto{
		TransmissionID:        transIdentity.ID,
		TransmissionSalt:      transIdentity.Salt,
		TransmissionRSA:       transIdentity.RSAPrivate.GetOldRSA(),
		ReceptionID:           receptionIdentity.ID,
		ReceptionSalt:         receptionIdentity.Salt,
		ReceptionRSA:          privateKey.GetOldRSA(),
		Precanned:             m.GetStorage().IsPrecanned(),
		RegistrationTimestamp: transIdentity.RegistrationTimestamp,
		RegCode:               regCode,
		TransmissionRegValidationSig: m.GetStorage().
			GetTransmissionRegistrationValidationSignature(),
		ReceptionRegValidationSig: m.GetStorage().
			GetReceptionRegistrationValidationSignature(),
		E2eDhPrivateKey: m.e2e.GetHistoricalDHPrivkey(),
		E2eDhPublicKey:  m.e2e.GetHistoricalDHPubkey(),
	}

	jsonBytes, err := json.Marshal(Usr)
	if err != nil {
		return nil, errors.WithMessage(err,
			"failed to register with permissioning")
	}

	return jsonBytes, nil
}

// GetAuth returns the auth.State.
func (m *E2e) GetAuth() auth.State {
	return m.auth
}

// GetE2E returns the e2e.Handler.
func (m *E2e) GetE2E() e2e.Handler {
	return m.e2e
}

// GetBackupContainer returns the backup Container.
func (m *E2e) GetBackupContainer() *Container {
	return m.backup
}

// DeleteContact removes a partner from E2e's storage.
func (m *E2e) DeleteContact(partnerId *id.ID) error {
	jww.DEBUG.Printf("Deleting contact with ID %s", partnerId)

	_, err := m.e2e.GetPartner(partnerId)
	if err != nil {
		return errors.WithMessagef(err, "Could not delete %s because "+
			"they could not be found", partnerId)
	}

	if err = m.e2e.DeletePartner(partnerId); err != nil {
		return err
	}

	m.backup.TriggerBackup("contact deleted")

	// FIXME: Do we need this?
	// c.e2e.Conversations().Delete(partnerId)

	// call delete requests to make sure nothing is lingering.
	// this is for safety to ensure the contact can be re-added
	// in the future
	_ = m.auth.DeleteRequest(partnerId)

	return nil
}

// DeleteContactNotify removes a partner from E2e's storage and sends an E2E
// message to the contact notifying them.
func (m *E2e) DeleteContactNotify(partnerId *id.ID, params e2e.Params) error {
	jww.DEBUG.Printf("Deleting contact with ID %s", partnerId)

	_, err := m.e2e.GetPartner(partnerId)
	if err != nil {
		return errors.WithMessagef(err, "Could not delete %s because "+
			"they could not be found", partnerId)
	}

	if err = m.e2e.DeletePartnerNotify(partnerId, params); err != nil {
		return err
	}

	m.backup.TriggerBackup("contact deleted")

	// FIXME: Do we need this?
	// c.e2e.Conversations().Delete(partnerId)

	// call delete requests to make sure nothing is lingering.
	// this is for safety to ensure the contact can be re-added
	// in the future
	_ = m.auth.DeleteRequest(partnerId)

	return nil
}

// MakeAuthCallbacksAdapter creates an authCallbacksAdapter.
func MakeAuthCallbacksAdapter(ac AuthCallbacks, e2e *E2e) *authCallbacksAdapter {
	return &authCallbacksAdapter{
		ac:  ac,
		e2e: e2e,
	}
}

// authCallbacksAdapter is an adapter type to make the AuthCallbacks type
// compatible with the auth.Callbacks type.
type authCallbacksAdapter struct {
	ac  AuthCallbacks
	e2e *E2e
}

// MakeAuthCB generates a new auth.Callbacks with the given AuthCallbacks.
func MakeAuthCB(e2e *E2e, cbs AuthCallbacks) auth.Callbacks {
	return &authCallbacksAdapter{
		ac:  cbs,
		e2e: e2e,
	}
}

// Request will be called when an auth Request message is processed.
func (aca *authCallbacksAdapter) Request(partner contact.Contact,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {
	aca.ac.Request(partner, receptionID, round, aca.e2e)
}

// Confirm will be called when an auth Confirm message is processed.
func (aca *authCallbacksAdapter) Confirm(partner contact.Contact,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {
	aca.ac.Confirm(partner, receptionID, round, aca.e2e)
}

// Reset will be called when an auth Reset operation occurs.
func (aca *authCallbacksAdapter) Reset(partner contact.Contact,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {
	aca.ac.Reset(partner, receptionID, round, aca.e2e)
}

// DefaultAuthCallbacks is a simple structure for providing a default
// AuthCallbacks implementation. It should generally not be used.
type DefaultAuthCallbacks struct{}

// Request will be called when an auth Request message is processed.
func (a DefaultAuthCallbacks) Request(contact.Contact,
	receptionID.EphemeralIdentity, rounds.Round, *E2e) {
	jww.ERROR.Printf("No valid auth callback assigned!")
}

// Confirm will be called when an auth Confirm message is processed.
func (a DefaultAuthCallbacks) Confirm(contact.Contact,
	receptionID.EphemeralIdentity, rounds.Round, *E2e) {
	jww.ERROR.Printf("No valid auth callback assigned!")
}

// Reset will be called when an auth Reset operation occurs.
func (a DefaultAuthCallbacks) Reset(contact.Contact,
	receptionID.EphemeralIdentity, rounds.Round, *E2e) {
	jww.ERROR.Printf("No valid auth callback assigned!")
}
