package ud

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/api"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/single"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/comms/client"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"math"
	"time"
)

type SingleInterface interface {
	TransmitSingleUse(contact.Contact, []byte, string, uint8, single.ReplyComm,
		time.Duration) error
	StartProcesses() (stoppable.Stoppable, error)
}

type Manager struct {
	// External
	client  *api.Client
	comms   *client.Comms
	rng     *fastRNG.StreamGenerator
	sw      interfaces.Switchboard
	storage *storage.Session
	net     interfaces.NetworkManager

	// Loaded from external access
	privKey *rsa.PrivateKey
	grp     *cyclic.Group

	// internal structures
	single SingleInterface
	myID   *id.ID

	// alternate User discovery service to circumvent production
	alternativeUd *alternateUd

	registered *uint32
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
func NewManager(client *api.Client, single *single.Manager) (*Manager, error) {
	jww.INFO.Println("ud.NewManager()")
	if client.NetworkFollowerStatus() != api.Running {
		return nil, errors.New(
			"cannot start UD State when network follower is not running.")
	}

	m := &Manager{
		client:  client,
		comms:   client.GetComms(),
		rng:     client.GetRng(),
		sw:      client.GetSwitchboard(),
		storage: client.GetStorage(),
		net:     client.GetNetworkInterface(),
		single:  single,
	}

	// check that user discovery is available in the NDF
	def := m.net.GetInstance().GetPartialNdf().Get()

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

	m.myID = m.storage.User().GetCryptographicIdentity().GetReceptionID()

	// get the commonly used data from storage
	m.privKey = m.storage.GetUser().ReceptionRSA

	// Load if the client is registered
	m.loadRegistered()

	// Store the pointer to the group locally for easy access
	m.grp = m.storage.E2e().GetGroup()

	return m, nil
}

// SetAlternativeUserDiscovery sets the alternativeUd object within manager.
// Once set, any user discovery operation will go through the alternative
// user discovery service.
// To undo this operation, use UnsetAlternativeUserDiscovery.
func (m *Manager) SetAlternativeUserDiscovery(altCert, altAddress, contactFile []byte) error {
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

// BackUpMissingFacts adds a registered fact to the Store object. It can take in both an
// email and a phone number. One or the other may be nil, however both is considered
// an error. It checks for the proper fact type for the associated fact.
// Any other fact.FactType is not accepted and returns an error and nothing is backed up.
// If you attempt to back up a fact type that has already been backed up,
// an error will be returned and nothing will be backed up.
// Otherwise, it adds the fact and returns whether the Store saved successfully.
func (m *Manager) BackUpMissingFacts(email, phone fact.Fact) error {
	return m.storage.GetUd().BackUpMissingFacts(email, phone)
}

// GetFacts returns a list of fact.Fact objects that exist within the
// Store's registeredFacts map.
func (m *Manager) GetFacts() []fact.Fact {
	return m.storage.GetUd().GetFacts()
}

// GetStringifiedFacts returns a list of stringified facts from the Store's
// registeredFacts map.
func (m *Manager) GetStringifiedFacts() []string {
	return m.storage.GetUd().GetStringifiedFacts()
}

// getHost returns the current UD host for the UD ID found in the NDF. If the
// host does not exist, then it is added and returned
func (m *Manager) getHost() (*connect.Host, error) {
	// Return alternative User discovery service if it has been set
	if m.alternativeUd != nil {
		return m.alternativeUd.host, nil
	}

	netDef := m.net.GetInstance().GetPartialNdf().Get()
	// Unmarshal UD ID from the NDF
	udID, err := id.Unmarshal(netDef.UDB.ID)
	if err != nil {
		return nil, errors.Errorf("failed to unmarshal UD ID from NDF: %+v", err)
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
		return nil, errors.WithMessage(err, "User Discovery host object could "+
			"not be constructed.")
	}

	return host, nil
}

// getContact returns the contact for UD as retrieved from the NDF.
func (m *Manager) getContact() (contact.Contact, error) {
	// Return alternative User discovery contact if set
	if m.alternativeUd != nil {
		// Unmarshal UD DH public key
		alternativeDhPubKey := m.storage.E2e().GetGroup().NewInt(1)
		if err := alternativeDhPubKey.UnmarshalJSON(m.alternativeUd.dhPubKey); err != nil {
			return contact.Contact{},
				errors.WithMessage(err, "Failed to unmarshal UD DH public key.")
		}

		return contact.Contact{
			ID:             m.alternativeUd.host.GetId(),
			DhPubKey:       alternativeDhPubKey,
			OwnershipProof: nil,
			Facts:          nil,
		}, nil
	}

	netDef := m.net.GetInstance().GetPartialNdf().Get()

	// Unmarshal UD ID from the NDF
	udID, err := id.Unmarshal(netDef.UDB.ID)
	if err != nil {
		return contact.Contact{},
			errors.Errorf("failed to unmarshal UD ID from NDF: %+v", err)
	}

	// Unmarshal UD DH public key
	dhPubKey := m.storage.E2e().GetGroup().NewInt(1)
	if err = dhPubKey.UnmarshalJSON(netDef.UDB.DhPubKey); err != nil {
		return contact.Contact{},
			errors.WithMessage(err, "Failed to unmarshal UD DH public key.")
	}

	return contact.Contact{
		ID:             udID,
		DhPubKey:       dhPubKey,
		OwnershipProof: nil,
		Facts:          nil,
	}, nil
}
