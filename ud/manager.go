package ud

import (
	"gitlab.com/elixxir/crypto/fastRNG"
	"sync"
	"time"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/event"
	"gitlab.com/elixxir/client/storage/versioned"
	store "gitlab.com/elixxir/client/ud/store"
	"gitlab.com/elixxir/client/xxdk"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
)

// Manager is the control structure for the contacting the user discovery service.
type Manager struct {

	// user is a sub-interface of the e2e.Handler. It allows the Manager
	// to retrieve the client's E2E information.
	user udE2e

	// store is an instantiation of this package's storage object.
	// It contains the facts that are in some state of being registered
	// with the UD service
	store *store.Store

	// comms is a sub-interface of the client.Comms interface. It contains
	// gRPC functions for registering and fact operations.
	comms Comms

	// factMux is to be used for Add/Remove fact.Fact operations.
	// This prevents simultaneous calls to Add/Remove calls which
	// may cause unexpected behaviour.
	factMux sync.Mutex

	// alternativeUd is an alternate User discovery service to circumvent
	// production. This is for testing with a separately deployed UD service.
	alternativeUd *alternateUd
}

// LoadOrNewManager loads an existing Manager from storage or creates a
// new one if there is no extant storage information.
//
// Params
//  - user is an interface that adheres to the xxdk.E2e object.
//  - comms is an interface that adheres to client.Comms object.
//  - follower is a method off of xxdk.Cmix which returns the network follower's status.
//  - username is the name of the user as it is registered with UD. This will be what the end user
//  provides if through the bindings.
//  - networkValidationSig is a signature provided by the network (i.e. the client registrar). This may
//  be nil, however UD may return an error in some cases (e.g. in a production level environment).
func LoadOrNewManager(user udE2e, comms Comms, follower udNetworkStatus,
	username string, networkValidationSig []byte) (*Manager, error) {
	jww.INFO.Println("ud.LoadOrNewManager()")

	// Construct manager
	m, err := loadOrNewManager(user, comms, follower)
	if err != nil {
		return nil, err
	}

	// Register manager
	rng := m.getRng().GetStream()
	defer rng.Close()
	err = m.register(username, networkValidationSig, rng, comms)
	if err != nil {
		return nil, err
	}

	return m, nil
}

// NewManagerFromBackup builds a new user discover manager from a backup.
// It will construct a manager that is already registered and restore
// already registered facts into store.
func NewManagerFromBackup(user udE2e, comms Comms, follower udNetworkStatus,
	email, phone fact.Fact) (*Manager, error) {
	jww.INFO.Println("ud.NewManagerFromBackup()")
	if follower() != xxdk.Running {
		return nil, errors.New(
			"cannot start UD Manager when " +
				"network follower is not running.")
	}

	// Initialize manager
	m := &Manager{
		user:  user,
		comms: comms,
	}

	// Initialize our store
	var err error
	m.store, err = store.NewOrLoadStore(m.getKv())
	if err != nil {
		return nil, err
	}

	// Put any passed in missing facts into store
	err = m.store.BackUpMissingFacts(email, phone)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to restore UD store "+
			"from backup")
	}

	// Set as registered. Since it's from a backup,
	// the user is already registered
	if err = setRegistered(m.getKv()); err != nil {
		return nil, errors.WithMessage(err, "failed to set client as "+
			"registered with user discovery.")
	}

	// Create the user discovery host object
	_, err = m.getOrAddUdHost()
	if err != nil {
		return nil, errors.WithMessage(err, "User Discovery host object could "+
			"not be constructed.")
	}

	return m, nil
}

// LoadOrNewAlternateUserDiscovery loads an existing Manager from storage or creates a
// new one if there is no extant storage information. This is different from LoadOrNewManager
// in that it allows the user to provide alternate User Discovery contact information.
// These parameters may be used to contact a separate UD server than the one run by the
// xx network team, one the user or a third-party may operate.
//
// Params
//  - user is an interface that adheres to the xxdk.E2e object.
//  - comms is an interface that adheres to client.Comms object.
//  - follower is a method off of xxdk.Cmix which returns the network follower's status.
//  - username is the name of the user as it is registered with UD. This will be what the end user
//  provides if through the bindings.
//  - networkValidationSig is a signature provided by the network (i.e. the client registrar). This may
//  be nil, however UD may return an error in some cases (e.g. in a production level environment).
//  - altCert is the TLS certificate for the alternate UD server.
//  - altAddress is the IP address of the alternate UD server.
//  - marshalledContact is the data within a marshalled contact.Contact.
//
// Returns
//  - A Manager object which is registered to the specified alternate UD service.
func LoadOrNewAlternateUserDiscovery(user udE2e, comms Comms, follower udNetworkStatus,
	username string, networkValidationSig []byte, altCert, altAddress,
	marshalledContact []byte) (*Manager, error) {

	jww.INFO.Println("ud.LoadOrNewAlternateUserDiscovery()")

	// Construct manager
	m, err := loadOrNewManager(user, comms, follower)
	if err != nil {
		return nil, err
	}

	// Set alternative user discovery
	err = m.setAlternateUserDiscovery(altCert, altAddress, marshalledContact)
	if err != nil {
		return nil, err
	}

	// Register manager
	rng := m.getRng().GetStream()
	defer rng.Close()
	err = m.register(username, networkValidationSig, rng, comms)
	if err != nil {
		return nil, err
	}

	return m, nil
}

// InitStoreFromBackup initializes the UD storage from the backup subsystem.
func InitStoreFromBackup(kv *versioned.KV,
	username, email, phone fact.Fact) error {
	// Initialize our store
	udStore, err := store.NewOrLoadStore(kv)
	if err != nil {
		return err
	}

	// Put any passed in missing facts into store
	err = udStore.BackUpMissingFacts(email, phone)
	if err != nil {
		return errors.WithMessage(err, "Failed to restore UD store "+
			"from backup")
	}

	// Set as registered. Since it's from a backup,
	// the user is already registered
	if err = setRegistered(kv); err != nil {
		return errors.WithMessage(err, "failed to set client as "+
			"registered with user discovery.")
	}

	return nil
}

// GetFacts returns a list of fact.Fact objects that exist within the
// Store's registeredFacts map.
func (m *Manager) GetFacts() fact.FactList {
	return m.store.GetFacts()
}

// GetStringifiedFacts returns a list of stringified facts from the Store's
// registeredFacts map.
func (m *Manager) GetStringifiedFacts() []string {
	return m.store.GetStringifiedFacts()
}

// GetContact returns the contact for UD as retrieved from the NDF.
func (m *Manager) GetContact() (contact.Contact, error) {
	grp, err := m.user.GetReceptionIdentity().GetGroup()
	if err != nil {
		return contact.Contact{}, err
	}
	// Return alternative User discovery contact if set
	if m.alternativeUd != nil {
		// Unmarshal UD DH public key
		alternativeDhPubKey := grp.NewInt(1)
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

	netDef := m.getCmix().GetInstance().GetPartialNdf().Get()

	// Unmarshal UD ID from the NDF
	udID, err := id.Unmarshal(netDef.UDB.ID)
	if err != nil {
		return contact.Contact{},
			errors.Errorf("failed to unmarshal UD ID from NDF: %+v", err)
	}

	// Unmarshal UD DH public key
	dhPubKey := grp.NewInt(1)
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

	netDef := m.getCmix().GetInstance().GetPartialNdf().Get()
	if netDef.UDB.Cert == "" {
		return nil, errors.New("NDF does not have User Discovery information, " +
			"is there network access?: Cert not present.")
	}

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

// loadOrNewManager is a helper function which loads from storage or
// creates a new Manager object.
func loadOrNewManager(user udE2e, comms Comms,
	follower udNetworkStatus) (*Manager, error) {
	if follower() != xxdk.Running {
		return nil, errors.New(
			"cannot start UD Manager when network follower is not running.")
	}

	// Initialize manager
	m := &Manager{
		user:  user,
		comms: comms,
	}

	if m.isRegistered() {
		// Load manager if already registered
		var err error
		m.store, err = store.NewOrLoadStore(m.getKv())
		if err != nil {
			return nil, errors.Errorf("Failed to initialize store: %v", err)
		}
		return m, nil
	}

	// Initialize store
	var err error
	m.store, err = store.NewOrLoadStore(m.getKv())
	if err != nil {
		return nil, errors.Errorf("Failed to initialize store: %v", err)
	}

	return m, nil
}

////////////////////////////////////////////////////////////////////////////////
// Internal Getters                                                           //
////////////////////////////////////////////////////////////////////////////////

// getCmix retrieve a sub-interface of cmix.Client.
// It allows the Manager to retrieve network state.
func (m *Manager) getCmix() udCmix {
	return m.user.GetCmix()
}

// getKv returns a versioned.KV used for isRegistered and setRegistered.
// This is separated from store operations as store's kv
// has a different prefix which breaks backwards compatibility.
func (m *Manager) getKv() *versioned.KV {
	return m.user.GetStorage().GetKV()
}

// getEventReporter returns an event.Reporter. This allows
// the Manager to report events to the other levels of the client.
func (m *Manager) getEventReporter() event.Reporter {
	return m.user.GetEventReporter()
}

// getRng returns a fastRNG.StreamGenerator. This RNG is for
// generating signatures for adding/removing facts.
func (m *Manager) getRng() *fastRNG.StreamGenerator {
	return m.user.GetRng()
}
