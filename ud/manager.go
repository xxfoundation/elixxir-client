package ud

import (
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/api"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/client/event"
	"gitlab.com/elixxir/client/interfaces/user"
	"gitlab.com/elixxir/client/single"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/comms/client"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

type SingleInterface interface {
	TransmitRequest(recipient contact.Contact, tag string, payload []byte,
		callback single.Response, param single.RequestParams, net cmix.Client, rng csprng.Source,
		e2eGrp *cyclic.Group) (id.Round, receptionID.EphemeralIdentity, error)
	StartProcesses() (stoppable.Stoppable, error)
}

type Userinfo interface {
	PortableUserInfo() user.Info
	GetUsername() (string, error)
	GetReceptionRegistrationValidationSignature() []byte
}

const (
// todo: populate with err messages
)

// todo: newuserDiscRegistratration, loadUserDiscRegistration
//  neworLoad?
// fixme: search/lookup off ud object
//  shouldn't be, pass stuff into
//

// ud takes an interface to backup to store dep loop

type Manager struct {
	// refactored
	// todo: docsting on what it is, why it's needed. For all things
	//  in this object and the object itself
	services cmix.Client
	e2e      e2e.Handler
	events   event.Manager
	store    *store.Store

	// todo: find a way to remove this, maybe just pass user into object (?)
	user Userinfo

	comms Comms
	rng   *fastRNG.StreamGenerator

	kv *versioned.KV

	// Loaded from external access
	privKey *rsa.PrivateKey
	grp     *cyclic.Group

	// internal structures
	myID *id.ID

	// alternate User discovery service to circumvent production
	alternativeUd *alternateUd
}

// alternateUd is an alternative user discovery service.
// This is used for testing, so client can avoid using
// the production server.
type alternateUd struct {
	host     *connect.Host
	dhPubKey []byte
}

// NewManager builds a new user discovery manager. It requires that an updated
// NDF is available and will error if one is not.
// todo: docstring, organize the order of arguments in a meaningful way
func NewManager(services cmix.Client, e2e e2e.Handler, events event.Manager,
	comms Comms, userStore Userinfo, rng *fastRNG.StreamGenerator,
	privKey *rsa.PrivateKey, username string,
	kv *versioned.KV) (*Manager, error) {
	jww.INFO.Println("ud.NewManager()")

	// fixme: figuring out a way to avoid importing api would be nice
	if client.NetworkFollowerStatus() != api.Running {
		return nil, errors.New(
			"cannot start UD Manager when network follower is not running.")
	}

	udStore, err := store.NewOrLoadStore(kv)
	if err != nil {
		return nil, errors.Errorf("Failed to initialize store: %v", err)
	}

	m := &Manager{
		services: services,
		e2e:      e2e,
		events:   events,
		comms:    comms,
		rng:      rng,
		store:    udStore,
		myID:     e2e.GetReceptionID(),
		grp:      e2e.GetGroup(),
		privKey:  privKey,
		user:     userStore,
		kv:       kv,
	}

	// check that user discovery is available in the NDF
	def := m.services.GetInstance().GetPartialNdf().Get()

	if def.UDB.Cert == "" {
		return nil, errors.New("NDF does not have User Discovery " +
			"information, is there network access?: Cert not present.")
	}

	// Pull user discovery ID from NDF
	udID, err := id.Unmarshal(def.UDB.ID)
	if err != nil {
		return nil, errors.Errorf("failed to unmarshal UD ID "+
			"from NDF: %+v", err)
	}

	udHost, err := m.getOrAddUdHost()
	if err != nil {
		return nil, errors.WithMessage(err, "User Discovery host object could "+
			"not be constructed.")
	}

	// Register with user discovery
	err = m.register(username, comms, udHost)
	if err != nil {
		return nil, errors.Errorf("Failed to register: %v", err)
	}

	// Set storage to registered
	// todo: maybe we don't need this?
	if err = m.setRegistered(); err != nil && m.events != nil {
		m.events.Report(1, "UserDiscovery", "Registration",
			fmt.Sprintf("User Registered with UD: %+v",
				username))
	}

	return m, nil
}

func LoadManager(services cmix.Client, e2e e2e.Handler, events event.Manager,
	comms Comms, userStore Userinfo, rng *fastRNG.StreamGenerator,
	privKey *rsa.PrivateKey, kv *versioned.KV) (*Manager, error) {

	m := &Manager{
		services: services,
		e2e:      e2e,
		events:   events,
		comms:    comms,
		user:     userStore,
		rng:      rng,
		privKey:  privKey,
		kv:       kv,
	}

	if !m.isRegistered() {
		return nil, errors.Errorf("LoadManager could not detect that " +
			"the user has been registered. Has a manager been initiated before?")
	}

	udStore, err := store.NewOrLoadStore(kv)
	if err != nil {
		return nil, errors.Errorf("Failed to initialize store: %v", err)
	}

	m.store = udStore

	return m, err
}

// SetAlternativeUserDiscovery sets the alternativeUd object within manager.
// Once set, any user discovery operation will go through the alternative
// user discovery service.
// To undo this operation, use UnsetAlternativeUserDiscovery.
func (m *Manager) SetAlternativeUserDiscovery(altCert, altAddress,
	contactFile []byte) error {
	params := connect.GetDefaultHostParams()
	params.AuthEnabled = false

	udIdBytes, dhPubKey, err := contact.ReadContactFromFile(contactFile)
	if err != nil {
		return err
	}

	udID, err := id.Unmarshal(udIdBytes)
	if err != nil {
		return err
	}

	// Add a new host and return it if it does not already exist
	host, err := m.comms.AddHost(udID, string(altAddress),
		altCert, params)
	if err != nil {
		return errors.WithMessage(err, "User Discovery host object could "+
			"not be constructed.")
	}

	m.alternativeUd = &alternateUd{
		host:     host,
		dhPubKey: dhPubKey,
	}

	return nil
}

// UnsetAlternativeUserDiscovery clears out the information from
// the Manager object.
func (m *Manager) UnsetAlternativeUserDiscovery() error {
	if m.alternativeUd == nil {
		return errors.New("Alternative User Discovery is already unset.")
	}

	m.alternativeUd = nil
	return nil
}

// BackUpMissingFacts adds a registered fact to the Store object.
// It can take in both an email and a phone number. One or the other may be nil,
// however both is considered an error. It checks for the proper fact type for
// the associated fact. Any other fact.FactType is not accepted and returns an
// error and nothing is backed up. If you attempt to back up a fact type that h
// as already been backed up, an error will be returned and nothing will be
// backed up. Otherwise, it adds the fact and returns whether the Store saved
// successfully.
func (m *Manager) BackUpMissingFacts(email, phone fact.Fact) error {
	return m.store.BackUpMissingFacts(email, phone)
}

// GetFacts returns a list of fact.Fact objects that exist within the
// Store's registeredFacts map.
func (m *Manager) GetFacts() []fact.Fact {
	return m.store.GetFacts()
}

// GetStringifiedFacts returns a list of stringified facts from the Store's
// registeredFacts map.
func (m *Manager) GetStringifiedFacts() []string {
	return m.store.GetStringifiedFacts()
}

// GetContact returns the contact for UD as retrieved from the NDF.
func (m *Manager) GetContact() (contact.Contact, error) {
	// Return alternative User discovery contact if set
	if m.alternativeUd != nil {
		// Unmarshal UD DH public key
		alternativeDhPubKey := m.grp.NewInt(1)
		if err := alternativeDhPubKey.
			UnmarshalJSON(m.alternativeUd.dhPubKey); err != nil {
			return contact.Contact{},
				errors.WithMessage(err, "Failed to unmarshal UD "+
					"DH public key.")
		}

		return contact.Contact{
			ID:             m.alternativeUd.host.GetId(),
			DhPubKey:       alternativeDhPubKey,
			OwnershipProof: nil,
			Facts:          nil,
		}, nil
	}

	netDef := m.services.GetInstance().GetPartialNdf().Get()

	// Unmarshal UD ID from the NDF
	udID, err := id.Unmarshal(netDef.UDB.ID)
	if err != nil {
		return contact.Contact{},
			errors.Errorf("failed to unmarshal UD ID from NDF: %+v", err)
	}

	// Unmarshal UD DH public key
	dhPubKey := m.grp.NewInt(1)
	if err = dhPubKey.UnmarshalJSON(netDef.UDB.DhPubKey); err != nil {
		return contact.Contact{},
			errors.WithMessage(err, "Failed to unmarshal UD DH "+
				"public key.")
	}

	return contact.Contact{
		ID:             udID,
		DhPubKey:       dhPubKey,
		OwnershipProof: nil,
		Facts:          nil,
	}, nil
}

// getOrAddUdHost returns the current UD host for the UD ID found in the NDF.
// If the host does not exist, then it is added and returned.
func (m *Manager) getOrAddUdHost() (*connect.Host, error) {
	// Return alternative User discovery service if it has been set
	if m.alternativeUd != nil {
		return m.alternativeUd.host, nil
	}

	netDef := m.services.GetInstance().GetPartialNdf().Get()
	// Unmarshal UD ID from the NDF
	udID, err := id.Unmarshal(netDef.UDB.ID)
	if err != nil {
		return nil, errors.Errorf("failed to "+
			"unmarshal UD ID from NDF: %+v", err)
	}

	// Return the host, if it exists
	host, exists := m.comms.GetHost(udID)
	if exists {
		return host, nil
	}

	params := connect.GetDefaultHostParams()
	params.AuthEnabled = false
	params.SendTimeout = 20 * time.Second

	// Add a new host and return it if it does not already exist
	host, err = m.comms.AddHost(udID, netDef.UDB.Address,
		[]byte(netDef.UDB.Cert), params)
	if err != nil {
		return nil, errors.WithMessage(err, "User Discovery host "+
			"object could not be constructed.")
	}

	return host, nil
}
