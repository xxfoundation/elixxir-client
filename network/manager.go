////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package network

// manager.go controls access to network resources. Interprocess communications
// and intraclient state are accessible through the context object.

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/context"
	"gitlab.com/elixxir/comms/client"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/switchboard"
	"gitlab.com/xx_network/primitives/id"
	"sync"
	"time"
)

// Manager implements the NetworkManager interface inside context. It
// controls access to network resources and implements all of the communications
// functions used by the client.
type Manager struct {
	// Comms pointer to send/recv messages
	Comms *client.Comms
	// Context contains all of the keying info used to send messages
	Context *context.Context
}

// NewManager builds a new reception manager object using inputted key fields
func NewManager(context *context.Context, uid *id.ID, privKey, pubKey,
	salt []byte) (*Manager, error) {
	comms, err := client.NewClientComms(uid, pubKey, privKey, salt)
	if err != nil {
		return nil, err
	}

	cm := &Manager{
		Comms:   comms,
		Context: ctx,
	}

	return cm, nil
}

// GetRemoteVersion contacts the permissioning server and returns the current
// supported client version.
func (m *Manager) GetRemoteVersion() (string, error) {
	permissioningHost, ok := m.Comms.GetHost(&id.Permissioning)
	if !ok {
		return "", errors.Errorf("no permissioning host with id %s",
			id.Permissioning)
	}
	registrationVersion, err := m.Comms.SendGetCurrentClientVersionMessage(
		permissioningHost)
	if err != nil {
		return "", err
	}
	return registrationVersion.Version, nil
}
