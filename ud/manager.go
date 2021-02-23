package ud

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/api"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/contact"
	"gitlab.com/elixxir/client/single"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/comms/client"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

type SingleInterface interface {
	TransmitSingleUse(contact.Contact, []byte, string, uint8, single.ReplyComm,
		time.Duration) error
	StartProcesses() stoppable.Stoppable
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
	udContact contact.Contact
	privKey   *rsa.PrivateKey
	grp       *cyclic.Group

	// Internal maps
	host   *connect.Host
	single SingleInterface

	registered *uint32
}

// New manager builds a new user discovery manager. It requires that an
// updated NDF is available and will error if one is not.
func NewManager(client *api.Client, single *single.Manager) (*Manager, error) {
	jww.INFO.Println("ud.NewManager()")
	if !client.GetHealth().IsHealthy() {
		return nil, errors.New("cannot start UD Manager when network was " +
			"never healthy.")
	}

	m := &Manager{
		client:    client,
		comms:     client.GetComms(),
		rng:       client.GetRng(),
		sw:        client.GetSwitchboard(),
		storage:   client.GetStorage(),
		net:       client.GetNetworkInterface(),
		udContact: contact.Contact{},
		single:    single,
	}

	var err error

	// check that user discovery is available in the ndf
	def := m.net.GetInstance().GetPartialNdf().Get()
	if m.udContact.ID, err = id.Unmarshal(def.UDB.ID); err != nil {
		return nil, errors.WithMessage(err, "NDF does not have User Discovery "+
			"information; is there network access?: ID could not be "+
			"unmarshaled.")
	}

	if def.UDB.Cert == "" {
		return nil, errors.New("NDF does not have User Discovery information, " +
			"is there network access?: Cert not present.")
	}

	// Unmarshal UD DH public key
	m.udContact.DhPubKey = m.storage.E2e().GetGroup().NewInt(1)
	if err = m.udContact.DhPubKey.UnmarshalJSON(def.UDB.DhPubKey); err != nil {
		return nil, errors.WithMessage(err, "Failed to unmarshal UD DH public key.")
	}

	// Create the user discovery host object
	hp := connect.GetDefaultHostParams()
	m.host, err = m.comms.AddHost(&id.UDB, def.UDB.Address, []byte(def.UDB.Cert), hp)
	if err != nil {
		return nil, errors.WithMessage(err, "User Discovery host object could "+
			"not be constructed.")
	}

	// Get the commonly used data from storage
	m.privKey = m.storage.GetUser().ReceptionRSA

	// Load if the client is registered
	m.loadRegistered()

	// Store the pointer to the group locally for easy access
	m.grp = m.storage.E2e().GetGroup()

	return m, nil
}
