////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package ud

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/xx_network/comms/connect"
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

	// Unmarshal the new contact
	con, err := contact.Unmarshal(contactFile)
	if err != nil {
		return err
	}

	// Add a new host and return it if it does not already exist
	host, err := m.comms.AddHost(con.ID, address,
		cert, params)
	if err != nil {
		return errors.WithMessage(err, "User Discovery host object could "+
			"not be constructed.")
	}

	// Set the user discovery object within the manager
	m.ud = &userDiscovery{
		host: host,
		contact: contact.Contact{
			ID:       con.ID,
			DhPubKey: con.DhPubKey,
		},
	}

	return nil
}
