package ud

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
)

// userDiscovery is the user discovery's contact information.
type userDiscovery struct {
	host     *connect.Host
	dhPubKey []byte
}

// setUserDiscovery sets the ud object within Manager.
// The specified the contact information will be used for
// all further Manager operations which contact the UD server.
func (m *Manager) setUserDiscovery(altCert,
	contactFile []byte, altAddress string) error {
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
	host, err := m.comms.AddHost(udID, altAddress,
		altCert, params)
	if err != nil {
		return errors.WithMessage(err, "User Discovery host object could "+
			"not be constructed.")
	}

	m.ud = &userDiscovery{
		host:     host,
		dhPubKey: dhPubKey,
	}

	return nil
}
