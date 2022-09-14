package ud

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/event"
	"gitlab.com/elixxir/client/storage/versioned"
	store "gitlab.com/elixxir/client/ud/store"
	"gitlab.com/elixxir/client/xxdk"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/primitives/fact"
	"sync"
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

	// ud is the tracker for the contact information of the specified UD server.
	// This information is specified in Manager's constructors (NewOrLoad and NewManagerFromBackup).
	ud *userDiscovery

	// These objects handle username validation.
	// The validation signature is saved into usernameValidationSignature
	// on the first query and lazily loaded. The usernameValidationMux
	// handles asynchronous queries to get Manager.GetUsernameValidationSignature
	usernameValidationMux       sync.Mutex
	usernameValidationSignature []byte
}

// NewOrLoad loads an existing Manager from storage or creates a
// new one if there is no extant storage information. Parameters need be provided
// to specify how to connect to the User Discovery service. These parameters may be used
// to contact either the UD server hosted by the xx network team or a custom
// third-party operated server.
//
// Params
//  - user is an interface that adheres to the xxdk.E2e object.
//  - comms is an interface that adheres to client.Comms object.
//  - follower is a method off of xxdk.Cmix which returns the network follower's status.
//  - username is the name of the user as it is registered with UD. This will be what the end user
//    provides if through the bindings.
//  - networkValidationSig is a signature provided by the network (i.e. the client registrar). This may
//    be nil, however UD may return an error in some cases (e.g. in a production level environment).
//  - cert is the TLS certificate for the UD server this call will connect with.
//  - contactFile is the data within a marshalled contact.Contact. This represents the
//    contact file of the server this call will connect with.
//  - address is the IP address of the UD server this call will connect with.
//
// Returns
//  - A Manager object which is registered to the specified UD service.
func NewOrLoad(user udE2e, comms Comms, follower udNetworkStatus,
	username string, networkValidationSig,
	cert, contactFile []byte, address string) (*Manager, error) {

	jww.INFO.Println("ud.NewOrLoad()")

	if follower() != xxdk.Running {
		return nil, errors.New(
			"cannot start UD Manager when network follower is not running.")
	}

	// Initialize manager
	m := &Manager{
		user:  user,
		comms: comms,
	}

	// Set user discovery
	err := m.setUserDiscovery(cert, contactFile, address)
	if err != nil {
		return nil, err
	}

	// Initialize store
	m.store, err = store.NewOrLoadStore(m.getKv())
	if err != nil {
		return nil, errors.Errorf("Failed to initialize store: %v", err)
	}

	// If already registered, return
	if IsRegistered(m.getKv()) {
		return m, nil
	}

	// Register manager
	rng := m.getRng().GetStream()
	defer rng.Close()
	err = m.register(username, networkValidationSig, rng, comms)
	if err != nil {
		return nil, err
	}

	usernameFact, err := fact.NewFact(fact.Username, username)
	if err != nil {
		return nil, err
	}

	err = m.store.StoreUsername(usernameFact)

	return m, nil
}

// NewManagerFromBackup builds a new user discover manager from a backup.
// It will construct a manager that is already registered and restore
// already registered facts into store.
//
// Params
//  - user is an interface that adheres to the xxdk.E2e object.
//  - comms is an interface that adheres to client.Comms object.
//  - follower is a method off of xxdk.Cmix which returns the network follower's status.
//  - username is the name of the user as it is registered with UD. This will be what the end user
//    provides if through the bindings.
//  - networkValidationSig is a signature provided by the network (i.e. the client registrar). This may
//    be nil, however UD may return an error in some cases (e.g. in a production level environment).
//  - email is a fact.Fact (type Email) which has been registered with UD already.
//  - phone is a fact.Fact (type Phone) which has been registered with UD already.
//  - cert is the TLS certificate for the UD server this call will connect with.
//  - contactFile is the data within a marshalled contact.Contact. This represents the
//    contact file of the server this call will connect with.
//  - address is the IP address of the UD server this call will connect with.
//
// Returns
//  - A Manager object which is registered to the specified UD service.
func NewManagerFromBackup(user udE2e, comms Comms, follower udNetworkStatus,
	username, email, phone fact.Fact,
	cert, contactFile []byte, address string) (*Manager, error) {
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
	err = m.store.BackUpMissingFacts(username, email, phone)
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

	err = m.setUserDiscovery(cert, contactFile, address)
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
	err = udStore.BackUpMissingFacts(username, email, phone)
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

// GetContact returns the contact.Contact for UD.
func (m *Manager) GetContact() contact.Contact {
	return m.ud.contact
}

// GetUsername returns the username from the Manager's store.
func (m *Manager) GetUsername() (string, error) {
	return m.store.GetUsername()
}

////////////////////////////////////////////////////////////////////////////////
// Internal Getters                                                           //
////////////////////////////////////////////////////////////////////////////////

// getCmix retrieve a sub-interface of cmix.Client.
// It allows the Manager to retrieve network state.
func (m *Manager) getCmix() udCmix {
	return m.user.GetCmix()
}

// getKv returns a versioned.KV used for IsRegistered and setRegistered.
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
