package ud

import (
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/api"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/event"
	"gitlab.com/elixxir/client/storage/versioned"
	store "gitlab.com/elixxir/client/ud/store"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"math"
	"sync"
	"time"
)

// todo: docstring everything, go over everything. review existing
//  and rewrie
type Manager struct {
	// refactored
	// todo: docsting on what it is, why it's needed. For all things
	//  in this object and the object itself

	// Network is a sub-interface of the cmix.Client interface. It
	// allows the Manager to retrieve network state.
	network cmix.Client

	// e2e is a sub-interface of the e2e.Handler. It allows the Manager
	// to retrieve the client's E2E information.
	e2e E2E

	// events allows the Manager to report events to the other
	// levels of the client.
	events event.Reporter

	// store is an instantiation of this package's storage object.
	// It contains the facts that are in some state of being registered
	// with the UD service
	store *store.Store

	// user is a sub-interface of the user.User object in the storage package.
	// This allows the Manager to pull user information for registration
	// and verifying the client's identity
	user UserInfo

	// comms is a sub-interface of the client.Comms interface. It contains
	// gRPC functions for registering and fact operations.
	comms Comms

	// rng is a random number generator. This is used for cryptographic
	// signature operations.
	rng *fastRNG.StreamGenerator

	// kv is a versioned key-value store
	// fixme: this is used for isRegistered and setRegistered
	//  which should be moved to store if possible (prefixing might break this?)
	kv *versioned.KV

	// factMux is to be used for Add/Remove fact.Fact operations.
	// This prevents simultaneous calls to Add/Remove calls which
	// may cause unexpected behaviour.
	factMux sync.Mutex

	// alternativeUd is an alternate User discovery service to circumvent
	// production. This is for testing with a separately deployed UD service.
	alternativeUd *alternateUd
}

// NewManager builds a new user discovery manager.
// It requires that an updated
// NDF is available and will error if one is not.
func NewManager(services cmix.Client, e2e E2E,
	follower NetworkStatus,
	events event.Reporter, comms Comms, userStore UserInfo,
	rng *fastRNG.StreamGenerator, username string,
	kv *versioned.KV) (*Manager, error) {
	jww.INFO.Println("ud.NewManager()")

	if follower() != api.Running {
		return nil, errors.New(
			"cannot start UD Manager when network follower is not running.")
	}

	udStore, err := store.NewOrLoadStore(kv)
	if err != nil {
		return nil, errors.Errorf("Failed to initialize store: %v", err)
	}

	m := &Manager{
		network: services,
		e2e:     e2e,
		events:  events,
		comms:   comms,
		rng:     rng,
		store:   udStore,
		user:    userStore,
		kv:      kv,
	}

	// check that user discovery is available in the NDF
	def := m.network.GetInstance().GetPartialNdf().Get()

	if def.UDB.Cert == "" {
		return nil, errors.New("NDF does not have User Discovery " +
			"information, is there network access?: Cert not present.")
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

// NewManagerFromBackup builds a new user discover manager from a backup.
// It will construct a manager that is already registered and restore
// already registered facts into store.
func NewManagerFromBackup(services cmix.Client,
	e2e E2E, follower NetworkStatus,
	events event.Reporter, comms Comms,
	userStore UserInfo, rng *fastRNG.StreamGenerator,
	email, phone fact.Fact, kv *versioned.KV) (*Manager, error) {
	jww.INFO.Println("ud.NewManagerFromBackup()")
	if follower() != api.Running {
		return nil, errors.New(
			"cannot start UD Manager when " +
				"network follower is not running.")
	}

	m := &Manager{
		network: services,
		e2e:     e2e,
		events:  events,
		comms:   comms,
		user:    userStore,
		rng:     rng,
		kv:      kv,
	}

	udStore, err := store.NewOrLoadStore(kv)
	if err != nil {
		return nil, err
	}

	m.store = udStore

	err = m.store.BackUpMissingFacts(email, phone)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to restore UD store "+
			"from backup")
	}

	// check that user discovery is available in the NDF
	def := m.network.GetInstance().GetPartialNdf().Get()

	if def.UDB.Cert == "" {
		return nil, errors.New("NDF does not have User Discovery information, " +
			"is there network access?: Cert not present.")
	}

	// Create the user discovery host object
	hp := connect.GetDefaultHostParams()
	// Client will not send KeepAlive packets
	hp.KaClientOpts.Time = time.Duration(math.MaxInt64)
	hp.MaxRetries = 3
	hp.SendTimeout = 3 * time.Second
	hp.AuthEnabled = false

	// Set as registered. Since it's from a backup,
	// the client is already registered
	// todo: maybe we don't need this?
	if err = m.setRegistered(); err != nil {
		return nil, errors.WithMessage(err, "failed to set client as "+
			"registered with user discovery.")
	}

	return m, nil
}

func LoadManager(services cmix.Client, e2e E2E,
	events event.Reporter, comms Comms, userStore UserInfo,
	rng *fastRNG.StreamGenerator, kv *versioned.KV) (*Manager, error) {

	m := &Manager{
		network: services,
		e2e:     e2e,
		events:  events,
		comms:   comms,
		user:    userStore,
		rng:     rng,
		kv:      kv,
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
	grp := m.e2e.GetGroup()
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

	netDef := m.network.GetInstance().GetPartialNdf().Get()

	// Unmarshal UD ID from the NDF
	udID, err := id.Unmarshal(netDef.UDB.ID)
	if err != nil {
		return contact.Contact{},
			errors.Errorf("failed to unmarshal UD ID from NDF: %+v", err)
	}

	fmt.Printf("netDef ud dhpub: %v\n", netDef.UDB.DhPubKey)

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

	netDef := m.network.GetInstance().GetPartialNdf().Get()
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
