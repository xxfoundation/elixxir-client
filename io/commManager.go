////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Package io asynchronous sending functionality. This is managed by an outgoing
// messages channel and managed by the sender thread kicked off during
// initialization.
package io

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/comms/client"
	"sync"
	"time"
)

const PermissioningAddrID = "Permissioning"

type ConnAddr string

func (a ConnAddr) String() string {
	return string(a)
}

// CommManager implements the Communications interface
type CommManager struct {
	// Comms pointer to send/recv messages
	Comms *client.Comms

	nextId   func() []byte
	collator *Collator

	// blockTransmissions will use a mutex to prevent multiple threads from sending
	// messages at the same time.
	blockTransmissions bool
	// transmitDelay is the minimum delay between transmissions.
	transmitDelay time.Duration
	// Map that holds a record of the messages that this client successfully
	// received during this session
	receivedMessages   map[string]struct{}
	recievedMesageLock sync.RWMutex

	sendLock sync.Mutex

	lock sync.RWMutex
}

func NewCommManager() *CommManager {
	cm := &CommManager{
		nextId:             parse.IDCounter(),
		collator:           NewCollator(),
		blockTransmissions: true,
		transmitDelay:      1000 * time.Millisecond,
		receivedMessages:   make(map[string]struct{}),
		Comms:              &client.Comms{},
	}

	return cm
}

// Connects to the permissioning server, if we know about it, to get the latest
// version from it
func (cm *CommManager) GetRemoteVersion() (string, error) { // need this but make getremoteversion, handle versioning in client
	permissioningHost, ok := cm.Comms.GetHost(PermissioningAddrID)
	if !ok {
		return "", errors.Errorf("Failed to find permissioning host with id %s", PermissioningAddrID)
	}
	registrationVersion, err := cm.Comms.
		SendGetCurrentClientVersionMessage(permissioningHost)
	if err != nil {
		return "", errors.Wrap(err, "Couldn't get current version from permissioning")
	}
	return registrationVersion.Version, nil
}

func (cm *CommManager) DisableBlockingTransmission() { // flag passed into receiver
	cm.blockTransmissions = false
}

func (cm *CommManager) SetRateLimit(delay time.Duration) { // pass into received
	cm.transmitDelay = delay
}
