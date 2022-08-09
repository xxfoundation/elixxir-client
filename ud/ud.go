package ud

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

// userDiscovery is the user discovery's contact information.
type userDiscovery struct {
	host    *connect.Host
	contact contact.Contact
}

// setUserDiscovery sets the ud object within Manager.
// The specified the contact information will be used for
// all further Manager operations which contact the UD server.
func (m *Manager) setUserDiscovery(cert,
	contactFile []byte, address string) error {
	params := connect.GetDefaultHostParams()
	params.AuthEnabled = false
	params.SendTimeout = 20 * time.Second

	udIdBytes, dhPubKeyBytes, err := contact.ReadContactFromFile(contactFile)
	if err != nil {
		return err
	}

	udID, err := id.Unmarshal(udIdBytes)
	if err != nil {
		return err
	}

	// Add a new host and return it if it does not already exist
	host, err := m.comms.AddHost(udID, address,
		cert, params)
	if err != nil {
		return errors.WithMessage(err, "User Discovery host object could "+
			"not be constructed.")
	}

	dhPubKey := m.user.GetE2E().GetGroup().NewInt(1)
	err = dhPubKey.UnmarshalJSON(dhPubKeyBytes)
	if err != nil {
		return err
	}

	m.ud = &userDiscovery{
		host: host,
		contact: contact.Contact{
			ID:       udID,
			DhPubKey: dhPubKey,
		},
	}

	return nil
}
