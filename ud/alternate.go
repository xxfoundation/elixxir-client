package ud

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
)

// alternateUd is an alternative user discovery service.
// This is used for testing, so client can avoid contacting
// the production server. This requires an alternative,
// deployed UD service.
type alternateUd struct {
	host     *connect.Host
	dhPubKey []byte
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
