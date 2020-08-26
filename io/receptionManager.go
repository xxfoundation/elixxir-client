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
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/switchboard"
	"gitlab.com/xx_network/primitives/id"
	"sync"
	"time"
)

type ConnAddr string

func (a ConnAddr) String() string {
	return string(a)
}

// ReceptionManager implements the Communications interface
type ReceptionManager struct {
	// Comms pointer to send/recv messages
	Comms *client.Comms

	nextId   func() []byte
	collator *Collator

	//Flags if the network is using tls or note
	Tls bool
	// blockTransmissions will use a mutex to prevent multiple threads from sending
	// messages at the same time.
	blockTransmissions bool // pass into receiver
	// transmitDelay is the minimum delay between transmissions.
	transmitDelay time.Duration // same
	// Map that holds a record of the messages that this client successfully
	// received during this session
	receivedMessages   map[string]struct{}
	recievedMesageLock sync.RWMutex

	sendLock sync.Mutex

	// Buffer of messages that cannot be decrypted
	garbledMessages []*format.Message
	garbleLck       sync.Mutex

	switchboard *switchboard.Switchboard

	rekeyChan chan struct{}
	quitChan  chan struct{}
}

// Build a new reception manager object using inputted key fields
func NewReceptionManager(rekeyChan, quitChan chan struct{}, uid *id.ID,
	privKey, pubKey, salt []byte, switchb *switchboard.Switchboard) (
	*ReceptionManager, error) {
	comms, err := client.NewClientComms(uid, pubKey, privKey, salt)
	if err != nil {
		return nil, errors.Wrap(err,
			"Failed to get client comms using constructor: %+v")
	}

	cm := &ReceptionManager{
		nextId:             parse.IDCounter(),
		collator:           NewCollator(),
		blockTransmissions: true,
		transmitDelay:      1000 * time.Millisecond,
		receivedMessages:   make(map[string]struct{}),
		Comms:              comms,
		rekeyChan:          rekeyChan,
		quitChan:           quitChan,
		garbledMessages:    make([]*format.Message, 0),
		switchboard:        switchb,
		Tls:                true,
	}

	return cm, nil
}

// Connects to the permissioning server, if we know about it, to get the latest
// version from it
func (rm *ReceptionManager) GetRemoteVersion() (string, error) {
	permissioningHost, ok := rm.Comms.GetHost(&id.Permissioning)
	if !ok {
		return "", errors.Errorf("Failed to find permissioning host with id %s", id.Permissioning)
	}
	registrationVersion, err := rm.Comms.
		SendGetCurrentClientVersionMessage(permissioningHost)
	if err != nil {
		return "", errors.Wrap(err, "Couldn't get current version from permissioning")
	}
	return registrationVersion.Version, nil
}

func (rm *ReceptionManager) DisableBlockingTransmission() { // flag passed into receiver
	rm.blockTransmissions = false
}

func (rm *ReceptionManager) SetRateLimit(delay time.Duration) { // pass into received
	rm.transmitDelay = delay
}

// AppendGarbledMessage appends a message or messages to the garbled message
// buffer.
func (rm *ReceptionManager) AppendGarbledMessage(messages ...*format.Message) {
	rm.garbleLck.Lock()
	rm.garbledMessages = append(rm.garbledMessages, messages...)
	rm.garbleLck.Unlock()
}

// PopGarbledMessages returns the content of the garbled message buffer and
// deletes its contents.
func (rm *ReceptionManager) PopGarbledMessages() []*format.Message {
	rm.garbleLck.Lock()
	defer rm.garbleLck.Unlock()
	tempBuffer := rm.garbledMessages
	rm.garbledMessages = []*format.Message{}
	return tempBuffer
}

// GetSwitchboard returns the active switchboard for this reception manager
func (rm *ReceptionManager) GetSwitchboard() *switchboard.Switchboard {
	return rm.switchboard
}
