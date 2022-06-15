package e2eApi

import (
	"encoding/binary"
	"encoding/json"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"time"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/api"
	"gitlab.com/elixxir/client/auth"
	"gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/client/e2e/rekey"
	"gitlab.com/elixxir/client/storage/user"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/xx_network/primitives/id"
)

type Client struct {
	*api.Client
	auth        auth.State
	e2e         e2e.Handler
	backup      *Container
	e2eIdentity TransmissionIdentity
}

// Login creates a new e2eApi.Client backed by the api.Client persistent versioned.KV
// If identity == nil, a new TransmissionIdentity will be generated automagically
func Login(client *api.Client, callbacks auth.Callbacks,
	identity *TransmissionIdentity) (m *Client, err error) {
	return login(client, callbacks, identity, client.GetStorage().GetKV())
}

// LoginEphemeral creates a new e2eApi.Client backed by a totally ephemeral versioned.KV
// If identity == nil, a new TransmissionIdentity will be generated automagically
func LoginEphemeral(client *api.Client, callbacks auth.Callbacks,
	identity *TransmissionIdentity) (m *Client, err error) {
	return login(client, callbacks, identity, versioned.NewKV(ekv.MakeMemstore()))
}

// LoginLegacy creates a new e2eApi.Client backed by the api.Client persistent versioned.KV
// Uses the pre-generated transmission ID used by api.Client
// This function is designed to maintain backwards compatibility with previous xx messenger designs
// and should not be used for other purposes
func LoginLegacy(client *api.Client, callbacks auth.Callbacks) (m *Client, err error) {
	m = &Client{
		Client: client,
		backup: &Container{},
	}

	m.e2e, err = LoadOrInitE2e(client)
	if err != nil {
		return nil, err
	}

	m.auth, err = auth.NewState(client.GetStorage().GetKV(), client.GetCmix(),
		m.e2e, client.GetRng(), client.GetEventReporter(),
		auth.GetDefaultParams(), callbacks, m.backup.TriggerBackup)
	if err != nil {
		return nil, err
	}

	u := m.Client.GetUser()
	m.e2eIdentity = TransmissionIdentity{
		ID:            u.TransmissionID,
		RSAPrivatePem: u.TransmissionRSA,
		Salt:          u.TransmissionSalt,
		DHKeyPrivate:  u.E2eDhPrivateKey,
	}

	return m, err
}

// login creates a new e2eApi.Client backed by the given versioned.KV
func login(client *api.Client, callbacks auth.Callbacks,
	identity *TransmissionIdentity, kv *versioned.KV) (m *Client, err error) {
	e2eGrp := client.GetStorage().GetE2EGroup()

	// Create new identity automatically if one isn't specified
	if identity == nil {
		rng := client.GetRng().GetStream()
		newIdentity, err := MakeTransmissionIdentity(rng, e2eGrp)
		rng.Close()
		if err != nil {
			return nil, err
		}
		identity = &newIdentity
		client.GetCmix().AddIdentity(identity.ID, time.Time{}, !kv.IsMemStore())
	}

	m = &Client{
		Client:      client,
		backup:      &Container{},
		e2eIdentity: *identity,
	}

	//initialize the e2e storage
	err = e2e.Init(kv, identity.ID, identity.DHKeyPrivate, e2eGrp,
		rekey.GetDefaultEphemeralParams())
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

	m.auth, err = auth.NewState(kv, client.GetCmix(),
		m.e2e, client.GetRng(), client.GetEventReporter(),
		auth.GetDefaultTemporaryParams(), callbacks, m.backup.TriggerBackup)
	if err != nil {
		return nil, err
	}

	return m, err
}

// LoadOrInitE2e loads the e2e handler or makes a new one, generating a new
// e2e private key. It attempts to load via a legacy construction, then tries
// to load the modern one, creating a new modern ID if neither can be found
func LoadOrInitE2e(client *api.Client) (e2e.Handler, error) {
	usr := client.GetUser()
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
		//if no new e2e handler exists, initialize an e2e user
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

			client.GetCmix().AddIdentity(usr.ReceptionID, time.Time{}, true)
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

// GetUser replaces api.Client's GetUser with one which includes the e2e dh
// private keys
func (m *Client) GetUser() user.Info {
	u := m.Client.GetUser()
	u.E2eDhPrivateKey = m.e2e.GetHistoricalDHPrivkey()
	u.E2eDhPublicKey = m.e2e.GetHistoricalDHPubkey()
	return u
}

// GetTransmissionIdentity returns a safe copy of the Client TransmissionIdentity
func (m *Client) GetTransmissionIdentity() TransmissionIdentity {
	return m.e2eIdentity.DeepCopy()
}

// ConstructProtoUserFile is a helper function which is used for proto
// client testing.  This is used for development testing.
func (m *Client) ConstructProtoUserFile() ([]byte, error) {

	//load the registration code
	regCode, err := m.GetStorage().GetRegCode()
	if err != nil {
		return nil, errors.WithMessage(err, "failed to register with "+
			"permissioning")
	}

	Usr := user.Proto{
		TransmissionID:               m.GetUser().TransmissionID,
		TransmissionSalt:             m.GetUser().TransmissionSalt,
		TransmissionRSA:              m.GetUser().TransmissionRSA,
		ReceptionID:                  m.GetUser().ReceptionID,
		ReceptionSalt:                m.GetUser().ReceptionSalt,
		ReceptionRSA:                 m.GetUser().ReceptionRSA,
		Precanned:                    m.GetUser().Precanned,
		RegistrationTimestamp:        m.GetUser().RegistrationTimestamp,
		RegCode:                      regCode,
		TransmissionRegValidationSig: m.GetStorage().GetTransmissionRegistrationValidationSignature(),
		ReceptionRegValidationSig:    m.GetStorage().GetReceptionRegistrationValidationSignature(),
		E2eDhPrivateKey:              m.e2e.GetHistoricalDHPrivkey(),
		E2eDhPublicKey:               m.e2e.GetHistoricalDHPubkey(),
	}

	jsonBytes, err := json.Marshal(Usr)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to register with "+
			"permissioning")
	}

	return jsonBytes, nil
}

func (m *Client) GetAuth() auth.State {
	return m.auth
}

func (m *Client) GetE2E() e2e.Handler {
	return m.e2e
}

func (m *Client) GetBackupContainer() *Container {
	return m.backup
}

// DeleteContact is a function which removes a partner from Client's storage
func (m *Client) DeleteContact(partnerId *id.ID) error {
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
