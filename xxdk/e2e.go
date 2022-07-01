////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package xxdk

import (
	"encoding/binary"
	"encoding/json"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/auth"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/client/e2e/rekey"
	"gitlab.com/elixxir/client/storage/user"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/xx"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

// E2e object bundles a ReceptionIdentity with a Cmix
// and can be used for high level operations such as connections
type E2e struct {
	*Cmix
	auth        auth.State
	e2e         e2e.Handler
	backup      *Container
	e2eIdentity ReceptionIdentity
}

// AuthCallbacks is an adapter for the auth.Callbacks interface
// that allows for initializing an E2e object without an E2e-dependant auth.Callbacks
type AuthCallbacks interface {
	Request(partner contact.Contact, receptionID receptionID.EphemeralIdentity,
		round rounds.Round, e2e *E2e)
	Confirm(partner contact.Contact, receptionID receptionID.EphemeralIdentity,
		round rounds.Round, e2e *E2e)
	Reset(partner contact.Contact, receptionID receptionID.EphemeralIdentity,
		round rounds.Round, e2e *E2e)
}

// Login creates a new E2e backed by the xxdk.Cmix persistent versioned.KV
// It bundles a Cmix object with a ReceptionIdentity object
// and initializes the auth.State and e2e.Handler objects
func Login(client *Cmix, callbacks AuthCallbacks,
	identity ReceptionIdentity) (m *E2e, err error) {
	return login(client, callbacks, identity, client.GetStorage().GetKV())
}

// LoginEphemeral creates a new E2e backed by a totally ephemeral versioned.KV
func LoginEphemeral(client *Cmix, callbacks AuthCallbacks,
	identity ReceptionIdentity) (m *E2e, err error) {
	return login(client, callbacks, identity, versioned.NewKV(ekv.MakeMemstore()))
}

// LoginLegacy creates a new E2e backed by the xxdk.Cmix persistent versioned.KV
// Uses the pre-generated transmission ID used by xxdk.Cmix.
// This function is designed to maintain backwards compatibility with previous
// xx messenger designs and should not be used for other purposes.
func LoginLegacy(client *Cmix, callbacks AuthCallbacks) (m *E2e, err error) {
	m = &E2e{
		Cmix:   client,
		backup: &Container{},
	}

	m.e2e, err = loadOrInitE2eLegacy(client)
	if err != nil {
		return nil, err
	}

	userInfo := client.GetStorage().PortableUserInfo()
	client.GetCmix().AddIdentity(userInfo.ReceptionID, time.Time{}, true)

	err = client.AddService(m.e2e.StartProcesses)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to add "+
			"the e2e processies")
	}

	m.auth, err = auth.NewState(client.GetStorage().GetKV(), client.GetCmix(),
		m.e2e, client.GetRng(), client.GetEventReporter(),
		auth.GetDefaultParams(), MakeAuthCallbacksAdapter(callbacks, m), m.backup.TriggerBackup)
	if err != nil {
		return nil, err
	}

	m.e2eIdentity, err = buildReceptionIdentity(userInfo, m.e2e.GetGroup(), m.e2e.GetHistoricalDHPrivkey())
	return m, err
}

// LoginWithNewBaseNDF_UNSAFE initializes a client object from existing storage
// while replacing the base NDF.  This is designed for some specific deployment
// procedures and is generally unsafe.
func LoginWithNewBaseNDF_UNSAFE(storageDir string, password []byte,
	newBaseNdf string, params Params) (*E2e, error) {
	jww.INFO.Printf("LoginWithNewBaseNDF_UNSAFE()")

	def, err := ParseNDF(newBaseNdf)
	if err != nil {
		return nil, err
	}

	c, err := LoadCmix(storageDir, password, params)
	if err != nil {
		return nil, err
	}

	//store the updated base NDF
	c.storage.SetNDF(def)

	if def.Registration.Address != "" {
		err = c.initPermissioning(def)
		if err != nil {
			return nil, err
		}
	} else {
		jww.WARN.Printf("Registration with permissioning skipped due " +
			"to blank permissionign address. Cmix will not be " +
			"able to register or track network.")
	}

	err = c.registerFollower()
	if err != nil {
		return nil, err
	}

	return LoginLegacy(c, nil)
}

// LoginWithProtoClient creates a client object with a protoclient
// JSON containing the cryptographic primitives. This is designed for
// some specific deployment procedures and is generally unsafe.
func LoginWithProtoClient(storageDir string, password []byte,
	protoClientJSON []byte, newBaseNdf string, callbacks AuthCallbacks,
	params Params) (*E2e, error) {
	jww.INFO.Printf("LoginWithProtoClient()")

	def, err := ParseNDF(newBaseNdf)
	if err != nil {
		return nil, err
	}

	protoUser := &user.Proto{}
	err = json.Unmarshal(protoClientJSON, protoUser)
	if err != nil {
		return nil, err
	}

	err = NewProtoClient_Unsafe(newBaseNdf, storageDir, password,
		protoUser)
	if err != nil {
		return nil, err
	}

	c, err := LoadCmix(storageDir, password, params)
	if err != nil {
		return nil, err
	}

	c.storage.SetNDF(def)

	err = c.initPermissioning(def)
	if err != nil {
		return nil, err
	}

	err = c.registerFollower()
	if err != nil {
		return nil, err
	}

	userInfo := c.GetStorage().PortableUserInfo()
	receptionIdentity, err := buildReceptionIdentity(userInfo, c.GetStorage().GetE2EGroup(), protoUser.E2eDhPrivateKey)
	return Login(c, callbacks, receptionIdentity)
}

// login creates a new xxdk.E2e backed by the given versioned.KV
func login(client *Cmix, callbacks AuthCallbacks,
	identity ReceptionIdentity, kv *versioned.KV) (m *E2e, err error) {

	// Verify the passed-in ReceptionIdentity matches its properties
	privatePem, err := identity.GetRSAPrivatePem()
	if err != nil {
		return nil, err
	}
	generatedId, err := xx.NewID(privatePem.GetPublic(), identity.Salt, id.User)
	if err != nil {
		return nil, err
	}
	if !generatedId.Cmp(identity.ID) {
		return nil, errors.Errorf("Given identity %s is invalid, generated ID does not match",
			identity.ID.String())
	}

	e2eGrp := client.GetStorage().GetE2EGroup()
	m = &E2e{
		Cmix:        client,
		backup:      &Container{},
		e2eIdentity: identity,
	}

	client.network.AddIdentity(identity.ID, time.Time{}, true)

	//initialize the e2e storage
	dhPrivKey, err := identity.GetDHKeyPrivate()
	if err != nil {
		return nil, err
	}
	err = e2e.Init(kv, identity.ID, dhPrivKey, e2eGrp,
		rekey.GetDefaultEphemeralParams())
	if err != nil {
		return nil, err
	}

	// load or init the new e2e storage
	m.e2e, err = e2e.Load(kv,
		client.GetCmix(), identity.ID, e2eGrp, client.GetRng(),
		client.GetEventReporter())
	if err != nil {
		//initialize the e2e storage
		err = e2e.Init(kv, identity.ID, dhPrivKey, e2eGrp,
			rekey.GetDefaultParams())
		if err != nil {
			return nil, err
		}

		//load the new e2e storage
		m.e2e, err = e2e.Load(kv,
			client.GetCmix(), identity.ID, e2eGrp, client.GetRng(),
			client.GetEventReporter())
		if err != nil {
			return nil, errors.WithMessage(err, "Failed to load a "+
				"newly created e2e store")
		}
	}

	err = client.AddService(m.e2e.StartProcesses)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to add "+
			"the e2e processies")
	}

	m.auth, err = auth.NewState(kv, client.GetCmix(),
		m.e2e, client.GetRng(), client.GetEventReporter(),
		auth.GetDefaultTemporaryParams(), MakeAuthCallbacksAdapter(callbacks, m), m.backup.TriggerBackup)
	if err != nil {
		return nil, err
	}

	return m, err
}

// loadOrInitE2eLegacy loads the e2e handler or makes a new one, generating a new
// e2e private key. It attempts to load via a legacy construction, then tries
// to load the modern one, creating a new modern ID if neither can be found
func loadOrInitE2eLegacy(client *Cmix) (e2e.Handler, error) {
	usr := client.GetStorage().PortableUserInfo()
	e2eGrp := client.GetStorage().GetE2EGroup()
	kv := client.GetStorage().GetKV()

	//try to load a legacy e2e handler
	e2eHandler, err := e2e.LoadLegacy(kv,
		client.GetCmix(), usr.ReceptionID, e2eGrp, client.GetRng(),
		client.GetEventReporter(), rekey.GetDefaultParams())
	if err != nil {
		//if no legacy e2e handler exists, try to load a new one
		e2eHandler, err = e2e.Load(kv,
			client.GetCmix(), usr.ReceptionID, e2eGrp, client.GetRng(),
			client.GetEventReporter())
		if err != nil {
			jww.WARN.Printf("Failed to load e2e instance for %s, "+
				"creating a new one", usr.ReceptionID)

			//generate the key
			var privkey *cyclic.Int
			if client.GetStorage().IsPrecanned() {
				jww.WARN.Printf("Using Precanned DH key")
				precannedID := binary.BigEndian.Uint64(
					client.GetStorage().GetReceptionID()[:])
				privkey = generatePrecanDHKeypair(
					uint(precannedID),
					client.GetStorage().GetE2EGroup())
			} else if usr.E2eDhPrivateKey != nil {
				jww.INFO.Printf("Using pre-existing DH key")
				privkey = usr.E2eDhPrivateKey
			} else {
				jww.INFO.Printf("Generating new DH key")
				rngStream := client.GetRng().GetStream()
				privkey = diffieHellman.GeneratePrivateKey(
					len(e2eGrp.GetPBytes()),
					e2eGrp, rngStream)
				rngStream.Close()
			}

			//initialize the e2e storage
			err = e2e.Init(kv, usr.ReceptionID, privkey, e2eGrp,
				rekey.GetDefaultParams())
			if err != nil {
				return nil, err
			}

			//load the new e2e storage
			e2eHandler, err = e2e.Load(kv,
				client.GetCmix(), usr.ReceptionID, e2eGrp, client.GetRng(),
				client.GetEventReporter())
			if err != nil {
				return nil, errors.WithMessage(err, "Failed to load a "+
					"newly created e2e store")
			}
		} else {
			jww.INFO.Printf("Loaded a modern e2e instance for %s",
				usr.ReceptionID)
		}
	} else {
		jww.INFO.Printf("Loaded a legacy e2e instance for %s",
			usr.ReceptionID)
	}
	return e2eHandler, nil
}

// GetReceptionIdentity returns a safe copy of the E2e ReceptionIdentity
func (m *E2e) GetReceptionIdentity() ReceptionIdentity {
	return m.e2eIdentity.DeepCopy()
}

// ConstructProtoUserFile is a helper function which is used for proto
// client testing.  This is used for development testing.
func (m *E2e) ConstructProtoUserFile() ([]byte, error) {

	//load the registration code
	regCode, err := m.GetStorage().GetRegCode()
	if err != nil {
		return nil, errors.WithMessage(err, "failed to register with "+
			"permissioning")
	}

	transIdentity := m.Cmix.GetTransmissionIdentity()
	receptionIdentity := m.GetReceptionIdentity()
	privatePem, err := receptionIdentity.GetRSAPrivatePem()
	if err != nil {
		return nil, err
	}

	Usr := user.Proto{
		TransmissionID:        transIdentity.ID,
		TransmissionSalt:      transIdentity.Salt,
		TransmissionRSA:       transIdentity.RSAPrivatePem,
		ReceptionID:           receptionIdentity.ID,
		ReceptionSalt:         receptionIdentity.Salt,
		ReceptionRSA:          privatePem,
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
		return nil, errors.WithMessage(err, "failed to register with "+
			"permissioning")
	}

	return jsonBytes, nil
}

func (m *E2e) GetAuth() auth.State {
	return m.auth
}

func (m *E2e) GetE2E() e2e.Handler {
	return m.e2e
}

func (m *E2e) GetBackupContainer() *Container {
	return m.backup
}

// DeleteContact is a function which removes a partner from E2e's storage
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

// MakeAuthCallbacksAdapter creates an authCallbacksAdapter
func MakeAuthCallbacksAdapter(ac AuthCallbacks, e2e *E2e) *authCallbacksAdapter {
	return &authCallbacksAdapter{
		ac:  ac,
		e2e: e2e,
	}
}

// authCallbacksAdapter is an adapter type to make the AuthCallbacks type
// compatible with the auth.Callbacks type
type authCallbacksAdapter struct {
	ac  AuthCallbacks
	e2e *E2e
}

func (aca *authCallbacksAdapter) Request(partner contact.Contact,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {
	aca.ac.Request(partner, receptionID, round, aca.e2e)
}

func (aca *authCallbacksAdapter) Confirm(partner contact.Contact,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {
	aca.ac.Confirm(partner, receptionID, round, aca.e2e)
}

func (aca *authCallbacksAdapter) Reset(partner contact.Contact,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {
	aca.ac.Reset(partner, receptionID, round, aca.e2e)
}

// DefaultAuthCallbacks is a simple structure for providing a default Callbacks implementation
// It should generally not be used.
type DefaultAuthCallbacks struct{}

// Confirm will be called when an auth Confirm message is processed.
func (a DefaultAuthCallbacks) Confirm(contact.Contact,
	receptionID.EphemeralIdentity, rounds.Round, *E2e) {
	jww.ERROR.Printf("No valid auth callback assigned!")
}

// Request will be called when an auth Request message is processed.
func (a DefaultAuthCallbacks) Request(contact.Contact,
	receptionID.EphemeralIdentity, rounds.Round, *E2e) {
	jww.ERROR.Printf("No valid auth callback assigned!")
}

// Reset will be called when an auth Reset operation occurs.
func (a DefaultAuthCallbacks) Reset(contact.Contact,
	receptionID.EphemeralIdentity, rounds.Round, *E2e) {
	jww.ERROR.Printf("No valid auth callback assigned!")
}
