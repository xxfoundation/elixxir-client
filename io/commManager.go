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
	"sync/atomic"
	"time"
)

const maxAttempts = 15
const maxBackoffTime = 180
const PermissioningAddrID = "Permissioning"

type ConnAddr string

func (a ConnAddr) String() string {
	return string(a)
}

// CommManager implements the Communications interface
type CommManager struct {
	// Comms pointer to send/recv messages
	Comms *client.ClientComms

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
	//Callback which passes the connection status when it updates
	connectionStatusCallback ConnectionStatusCallback

	// blockTransmissions will use a mutex to prevent multiple threads from sending
	// messages at the same time.
	blockTransmissions bool
	// transmitDelay is the minimum delay between transmissions.
	transmitDelay time.Duration
	// Map that holds a record of the messages that this client successfully
	// received during this session
	receivedMessages map[string]struct{}

	sendLock sync.Mutex

	tryReconnect chan struct{}

	connectionStatus *uint32

	registrationVersion string

	lock sync.RWMutex
}

func NewCommManager(ndf *ndf.NetworkDefinition,
	callback ConnectionStatusCallback) *CommManager {

	status := uint32(0)

	cm := CommManager{
		nextId:                   parse.IDCounter(),
		collator:                 NewCollator(),
		blockTransmissions:       true,
		transmitDelay:            1000 * time.Millisecond,
		receivedMessages:         make(map[string]struct{}),
		Comms:                    &client.ClientComms{},
		tryReconnect:             make(chan struct{}),
		tls:                      true,
		ndf:                      ndf,
		receptionGatewayIndex:    len(ndf.Gateways) - 1,
		transmissionGatewayIndex: 0,
		connectionStatusCallback: callback,
		connectionStatus:         &status,
	}

	return &cm
}

// Connects to gateways using tls filepaths to create credential information
// for connection establishment
func (cm *CommManager) ConnectToGateways() error {
	var err error
	if len(cm.ndf.Gateways) < 1 {
		return errors.New("could not connect due to invalid number of nodes")
	}

	cm.setConnectionStatus(Connecting, 0)

	cm.Comms.ConnectionManager.SetMaxRetries(1)

	// connect to all gateways
	var wg sync.WaitGroup
	errChan := make(chan error, len(cm.ndf.Gateways))
	for i, gateway := range cm.ndf.Gateways {
		wg.Add(1)
		go func() {
			var gwCreds []byte

			cm.lock.RLock()
			if gateway.TlsCertificate != "" && cm.tls {
				gwCreds = []byte(gateway.TlsCertificate)
			}
			gwID := id.NewNodeFromBytes(cm.ndf.Nodes[i].ID).NewGateway()
			gwAddr := gateway.Address
			cm.lock.Unlock()

			globals.Log.INFO.Printf("Connecting to gateway %s at %s...",
				gwID.String(), gwAddr)
			err = cm.Comms.ConnectToRemote(gwID, gwAddr,
				gwCreds, false)

			if err != nil {
				errChan <- errors.New(fmt.Sprintf(
					"Failed to connect to gateway %s at %s: %+v",
					gwID.String(), gwAddr, err))
			}
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

	cm.setConnectionStatus(Online, 0)
	return nil
}

// Connects to the permissioning server, if we know about it, to get the latest
// version from it
func (cm *CommManager) UpdateRemoteVersion() error {
	registrationVersion, err := cm.Comms.
		SendGetCurrentClientVersionMessage(ConnAddr(PermissioningAddrID))
	if err != nil {
		return errors.Wrap(err, "Couldn't get current version from permissioning")
	}
	cm.registrationVersion = registrationVersion.Version
	return nil
}

func (cm *CommManager) GetConnectionCallback() ConnectionStatusCallback {
	return cm.connectionStatusCallback
}

//GetUpdatedNDF: Connects to the permissioning server to get the updated NDF from it
func (cm *CommManager) GetUpdatedNDF(currentNDF *ndf.NetworkDefinition) (*ndf.NetworkDefinition, error) {

	//Hash the client's ndf for comparison with registration's ndf
	hash := sha256.New()
	ndfBytes := currentNDF.Serialize()
	hash.Write(ndfBytes)
	ndfHash := hash.Sum(nil)

	//Put the hash in a message
	msg := &mixmessages.NDFHash{Hash: ndfHash}

	//Send the hash to registration
	response, err := cm.Comms.SendGetUpdatedNDF(ConnAddr(PermissioningAddrID), msg)
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
func (cm *CommManager) UpdateNDF(updatedNDF *ndf.NetworkDefinition) {
	cm.lock.Lock()
	cm.ndf = updatedNDF
	cm.receptionGatewayIndex = len(cm.ndf.Gateways) - 1
	cm.transmissionGatewayIndex = 0
	cm.lock.Unlock()
}

// Utility method, returns whether the local version and remote version are
// compatible
func (cm *CommManager) CheckVersion() (bool, error) {
	return checkVersion(globals.SEMVER, cm.registrationVersion)
}

// There's currently no need to keep connected to permissioning constantly,
// so we have functions to connect to and disconnect from it when a connection
// to permissioning is needed
func (cm *CommManager) ConnectToPermissioning() (connected bool, err error) {
	if cm.ndf.Registration.Address != "" {
		var regCert []byte
		if cm.ndf.Registration.TlsCertificate != "" && cm.tls {
			regCert = []byte(cm.ndf.Registration.TlsCertificate)
		}
		addr := ConnAddr(PermissioningAddrID)
		globals.Log.INFO.Printf("Connecting to permissioning/registration at %s...",
			cm.ndf.Registration.Address)
		err = cm.Comms.ConnectToRemote(addr, cm.ndf.Registration.Address, regCert, false)
		if err != nil {
			return false, errors.New(fmt.Sprintf(
				"Failed connecting to permissioning: %+v", err))
		}
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

func (cm *CommManager) DisconnectFromPermissioning() {
	globals.Log.DEBUG.Printf("Disconnecting from permissioning")
	cm.Comms.Disconnect(PermissioningAddrID)
}

func (cm *CommManager) Disconnect() {
	cm.Comms.DisconnectAll()
}

func (cm *CommManager) DisableTLS() {
	status := atomic.LoadUint32(cm.connectionStatus)

	if status != Setup {
		globals.Log.FATAL.Panicf("Cannot disable TLS" +
			"while communications are running")
	}
	cm.tls = false
}

func (cm *CommManager) GetRegistrationVersion() string {
	return cm.registrationVersion
}

func (cm *CommManager) DisableBlockingTransmission() {
	status := atomic.LoadUint32(cm.connectionStatus)

	if status != Setup {
		globals.Log.FATAL.Panicf("Cannot set tramsmission to blocking" +
			"while communications are running")
	}
	cm.blockTransmissions = false
}

func (cm *CommManager) SetRateLimit(delay time.Duration) {
	status := atomic.LoadUint32(cm.connectionStatus)

	if status != Setup {
		globals.Log.FATAL.Panicf("Cannot set the connection rate limit " +
			"while communications are running")
	}
	cm.transmitDelay = delay
}

func (cm *CommManager) GetConnectionStatus() uint32 {
	return atomic.LoadUint32(cm.connectionStatus)
}

func (cm *CommManager) setConnectionStatus(status uint32, timeout int) {
	atomic.SwapUint32(cm.connectionStatus, status)
	globals.Log.INFO.Printf("Connection status changed to: %v", status)
	go cm.connectionStatusCallback(status, timeout)
}

func toSeconds(duration time.Duration) int {
	return int(duration) / int(time.Second)
}
