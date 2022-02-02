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

	registered *uint32
}

// NewManager builds a new user discovery manager. It requires that an updated
// NDF is available and will error if one is not.
func NewManager(client *api.Client, single *single.Manager) (*Manager, error) {
	jww.INFO.Println("ud.NewManager()")
	if client.NetworkFollowerStatus() != api.Running {
		return nil, errors.New(
			"cannot start UD Manager when network follower is not running.")
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

	// Get the commonly used data from storage
	m.privKey = m.storage.GetUser().ReceptionRSA

	// Load if the client is registered
	m.loadRegistered()

	// Store the pointer to the group locally for easy access
	m.grp = m.storage.E2e().GetGroup()

	return m, nil
}

func (m *Manager) StoreFact(f fact.Fact) error {
	return m.storage.GetUd().StoreFact(f)
}

func (m *Manager) GetFacts() []fact.Fact {
	return m.storage.GetUd().GetFacts()
}

func (m *Manager) GetStringifiedFact() []string {
	return m.storage.GetUd().GetStringifiedFacts()
}

// getHost returns the current UD host for the UD ID found in the NDF. If the
// host does not exist, then it is added and returned
func (m *Manager) getHost() (*connect.Host, error) {
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
