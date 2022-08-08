package ud

import (
	"encoding/json"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/xx_network/comms/connect"
)

// alternateUd is an alternative user discovery service.
// This is used for testing, so client can avoid contacting
// the production server. This requires an alternative,
// deployed UD service.
type alternateUd struct {
	host     *connect.Host
	dhPubKey []byte
}

// setAlternateUserDiscovery sets the alternativeUd object within manager.
// Once set, any user discovery operation will go through the alternative
// user discovery service.
//
// To undo this operation, use UnsetAlternativeUserDiscovery.
func (m *Manager) setAlternateUserDiscovery(altCert, altAddress,
	contactFile []byte) error {
	params := connect.GetDefaultHostParams()
	params.AuthEnabled = false

	c := &contact.Contact{}
	err := json.Unmarshal(contactFile, c)
	if err != nil {
		return errors.Errorf("Failed to unmarshal contact file: %v", err)
	}

	// Add a new host and return it if it does not already exist
	host, err := m.comms.AddHost(c.ID, string(altAddress),
		altCert, params)
	if err != nil {
		return errors.WithMessage(err, "User Discovery host object could "+
			"not be constructed.")
	}

	dhPubJson, err := c.DhPubKey.MarshalJSON()
	if err != nil {
		return errors.Errorf("Failed to marshal Diffie-Helman public key: %v", err)
	}

	m.alternativeUd = &alternateUd{
		host:     host,
		dhPubKey: dhPubJson,
	}

	return nil
}

// UnsetAlternativeUserDiscovery clears out the information from
// the Manager object.
// fixme: I think this should be removed to avoid creating a Manager object
//  which has never been registered to production, and can't be w/o exporting
//  the Manger.register method.
func (m *Manager) UnsetAlternativeUserDiscovery() error {
	if m.alternativeUd == nil {
		return errors.New("Alternative User Discovery is already unset.")
	}

	m.alternativeUd = nil
	return nil
}
