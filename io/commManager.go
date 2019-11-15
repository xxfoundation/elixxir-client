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
	"gitlab.com/elixxir/comms/connect"
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

	//Defines network Topology
	ndf *ndf.NetworkDefinition
	//Flags if the network is using tls or note
	tls bool
	// Index in the NDF of the gateway used to receive messages
	receptionGatewayIndex int
	// Index in the NDF of the gateway used to send messages
	transmissionGatewayIndex int

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

	registrationVersion string

	lock sync.RWMutex
}

func NewCommManager(ndf *ndf.NetworkDefinition) *CommManager {
	cm := &CommManager{
		nextId:                   parse.IDCounter(),
		collator:                 NewCollator(),
		blockTransmissions:       true,
		transmitDelay:            1000 * time.Millisecond,
		receivedMessages:         make(map[string]struct{}),
		Comms:                    &client.Comms{},
		tls:                      true,
		ndf:                      ndf,
		receptionGatewayIndex:    len(ndf.Gateways) - 1,
		transmissionGatewayIndex: 0,
	}

	//cm.Comms.ConnectionManager.SetMaxRetries(1)

	return cm
}

// Connects to gateways using tls filepaths to create credential information
// for connection establishment
func (cm *CommManager) ConnectToGateways() error { // tear out
	var err error
	if len(cm.ndf.Gateways) < 1 {
		return errors.New("could not connect due to invalid number of nodes")
	}

	// connect to all gateways
	var wg sync.WaitGroup
	errChan := make(chan error, len(cm.ndf.Gateways))
	for i, gateway := range cm.ndf.Gateways {

		var gwCreds []byte

		cm.lock.RLock() // what is the purpose of this locked block
		if gateway.TlsCertificate != "" && cm.tls {
			gwCreds = []byte(gateway.TlsCertificate)
		}
		gwID := id.NewNodeFromBytes(cm.ndf.Nodes[i].ID).NewGateway()
		gwAddr := gateway.Address
		cm.lock.RUnlock()

		wg.Add(1)
		go func() { // Does this still need a thread?
			globals.Log.INFO.Printf("Connecting to gateway %s at %s...",
				gwID.String(), gwAddr)
			host, err := connect.NewHost(gwAddr, gwCreds, false)
			if err != nil {
				errChan <- errors.New(fmt.Sprintf(
					"Failed to create host for gateway %s at %s: %+v",
					gwID.String(), gwAddr, err))
			}
			cm.Comms.AddHost(gwID.String(), host)
			wg.Done()
		}()
		wg.Wait()

		var errs error
		for len(errChan) > 0 {
			err = <-errChan
			if errs != nil {
				errs = errors.Wrap(errs, err.Error())
			} else {
				errs = err
			}
		}

		if errs != nil {
			return errs
		}
	}

	return nil
}

// Connects to the permissioning server, if we know about it, to get the latest
// version from it
func (cm *CommManager) UpdateRemoteVersion() error { // need this but make getremoteversion, handle versioning in client
	permissioningHost, ok := cm.Comms.GetHost(PermissioningAddrID)
	if !ok {
		return errors.Errorf("Failed to find permissioning host with id %s", PermissioningAddrID)
	}
	registrationVersion, err := cm.Comms.
		SendGetCurrentClientVersionMessage(permissioningHost)
	if err != nil {
		return errors.Wrap(err, "Couldn't get current version from permissioning")
	}
	cm.registrationVersion = registrationVersion.Version
	return nil
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
		return cm.ndf, nil
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

// Update NDF modifies the network properties for the network which is
// communicated with
func (cm *CommManager) UpdateNDF(updatedNDF *ndf.NetworkDefinition) { // again, don't worry about ndf in this object
	cm.lock.Lock()
	cm.ndf = updatedNDF
	cm.receptionGatewayIndex = len(cm.ndf.Gateways) - 1
	cm.transmissionGatewayIndex = 0
	cm.lock.Unlock()
}

// Utility method, returns whether the local version and remote version are
// compatible
func (cm *CommManager) CheckVersion() (bool, error) { // again, version stuff, move to globals
	return checkVersion(globals.SEMVER, cm.registrationVersion)
}

// There's currently no need to keep connected to permissioning constantly,
// so we have functions to connect to and disconnect from it when a connection
// to permissioning is needed
func (cm *CommManager) ConnectToPermissioning() (connected bool, err error) { // this disappears, make host in simple call
	if cm.ndf.Registration.Address != "" {
		_, ok := cm.Comms.GetHost(PermissioningAddrID)
		if ok {
			return true, nil
		}
		var regCert []byte
		if cm.ndf.Registration.TlsCertificate != "" && cm.tls {
			regCert = []byte(cm.ndf.Registration.TlsCertificate)
		}

		globals.Log.INFO.Printf("Connecting to permissioning/registration at %s...",
			cm.ndf.Registration.Address)
		host, err := connect.NewHost(cm.ndf.Registration.Address, regCert, false)
		if err != nil {
			return false, errors.New(fmt.Sprintf(
				"Failed connecting to create host for permissioning: %+v", err))
		}
		cm.Comms.AddHost(PermissioningAddrID, host)

		globals.Log.INFO.Printf(
			"Connected to permissioning at %v successfully!",
			cm.ndf.Registration.Address)
		return true, nil
	} else {
		globals.Log.DEBUG.Printf("failed to connect to %v silently", cm.ndf.Registration.Address)
		// Without an NDF, we can't connect to permissioning, but this isn't an
		// error per se, because we should be phasing out permissioning at some
		// point
		return false, nil
	}
}

func (cm *CommManager) Disconnect() { // gone
	cm.Comms.DisconnectAll()
}

func (cm *CommManager) DisableTLS() { // gone
	cm.tls = false
}

func (cm *CommManager) GetRegistrationVersion() string { // on client
	return cm.registrationVersion
}

func (cm *CommManager) DisableBlockingTransmission() { // flag passed into receiver
	cm.blockTransmissions = false
}

func (cm *CommManager) SetRateLimit(delay time.Duration) { // pass into received
	cm.transmitDelay = delay
}
