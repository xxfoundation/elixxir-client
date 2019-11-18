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
	"crypto/sha256"
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/comms/client"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/ndf"
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

// Connects to gateways using tls filepaths to create credential information
// for connection establishment
func (cm *CommManager) AddGatewayHosts(ndf *ndf.NetworkDefinition) error { // tear out
	if len(ndf.Gateways) < 1 {
		return errors.New("could not connect due to invalid number of nodes")
	}

	// connect to all gateways
	var errs error = nil
	for i, gateway := range ndf.Gateways {

		var gwCreds []byte

		if gateway.TlsCertificate != "" {
			gwCreds = []byte(gateway.TlsCertificate)
		}
		gwID := id.NewNodeFromBytes(ndf.Nodes[i].ID).NewGateway()
		gwAddr := gateway.Address

		_, err := cm.Comms.AddHost(gwID.String(), gwAddr, gwCreds, false)
		if err != nil {
			err = errors.Errorf("Failed to create host for gateway %s at %s: %+v",
				gwID.String(), gwAddr, err)
			if errs != nil {
				errs = errors.Wrap(errs, err.Error())
			} else {
				errs = err
			}
		}
	}
	return errs
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

//GetUpdatedNDF: Connects to the permissioning server to get the updated NDF from it
func (cm *CommManager) GetUpdatedNDF(currentNDF *ndf.NetworkDefinition) (*ndf.NetworkDefinition, error) { // again, uses internal ndf.  stay here, return results instead

	//Hash the client's ndf for comparison with registration's ndf
	hash := sha256.New()
	ndfBytes := currentNDF.Serialize()
	hash.Write(ndfBytes)
	ndfHash := hash.Sum(nil)

	//Put the hash in a message
	msg := &mixmessages.NDFHash{Hash: ndfHash}

	host, ok := cm.Comms.GetHost(PermissioningAddrID)
	if !ok {
		return nil, errors.New("Failed to find permissioning host")
	}

	//Send the hash to registration
	response, err := cm.Comms.SendGetUpdatedNDF(host, msg)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to get ndf from permissioning: %v", err)
		return nil, errors.New(errMsg)
	}

	//If there was no error and the response is nil, client's ndf is up-to-date
	if response == nil {
		globals.Log.DEBUG.Printf("Client NDF up-to-date")
		return nil, nil
	}

	//FixMe: use verify instead? Probs need to add a signature to ndf, like in registration's getupdate?

	globals.Log.INFO.Printf("Remote NDF: %s", string(response.Ndf))

	//Otherwise pull the ndf out of the response
	updatedNdf, _, err := ndf.DecodeNDF(string(response.Ndf))
	if err != nil {
		//If there was an error decoding ndf
		errMsg := fmt.Sprintf("Failed to decode response to ndf: %v", err)
		return nil, errors.New(errMsg)
	}
	return updatedNdf, nil
}

// There's currently no need to keep connected to permissioning constantly,
// so we have functions to connect to and disconnect from it when a connection
// to permissioning is needed
func (cm *CommManager) AddPermissioningHost(reg *ndf.Registration) (bool, error) { // this disappears, make host in simple call
	if reg.Address != "" {
		_, ok := cm.Comms.GetHost(PermissioningAddrID)
		if ok {
			return true, nil
		}
		var regCert []byte
		if reg.TlsCertificate != "" {
			regCert = []byte(reg.TlsCertificate)
		}

		_, err := cm.Comms.AddHost(PermissioningAddrID, reg.Address, regCert, false)
		if err != nil {
			return false, errors.New(fmt.Sprintf(
				"Failed connecting to create host for permissioning: %+v", err))
		}
		return true, nil
	} else {
		globals.Log.DEBUG.Printf("failed to connect to %v silently", reg.Address)
		// Without an NDF, we can't connect to permissioning, but this isn't an
		// error per se, because we should be phasing out permissioning at some
		// point
		return false, nil
	}
}

func (cm *CommManager) DisableBlockingTransmission() { // flag passed into receiver
	cm.blockTransmissions = false
}

func (cm *CommManager) SetRateLimit(delay time.Duration) { // pass into received
	cm.transmitDelay = delay
}
