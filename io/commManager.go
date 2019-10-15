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
	"bytes"
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
	nextId   func() []byte
	collator *Collator

	//Defines network Topology
	ndf *ndf.NetworkDefinition
	//Flags if the network is using TLS or note
	TLS bool
	// Index in the NDF of the gateway used to receive messages
	ReceptionGatewayIndex int
	// Index in the NDF of the gateway used to send messages
	TransmissionGatewayIndex int
	//Callback which passes the connection status when it updates
	connectionStatusCallback ConnectionStatusCallback

	// BlockTransmissions will use a mutex to prevent multiple threads from sending
	// messages at the same time.
	BlockTransmissions bool
	// TransmitDelay is the minimum delay between transmissions.
	TransmitDelay time.Duration
	// Map that holds a record of the messages that this client successfully
	// received during this session
	ReceivedMessages map[string]struct{}
	// Comms pointer to send/recv messages
	Comms    *client.ClientComms
	sendLock sync.Mutex

	tryReconnect chan struct{}

	connectionStatus *uint32

	RegistrationVersion string

	lock sync.RWMutex
}

func NewCommManager(ndf *ndf.NetworkDefinition,
	callback ConnectionStatusCallback) *CommManager {

	status := uint32(0)

	cm := CommManager{
		nextId:                   parse.IDCounter(),
		collator:                 NewCollator(),
		BlockTransmissions:       true,
		TransmitDelay:            1000 * time.Millisecond,
		ReceivedMessages:         make(map[string]struct{}),
		Comms:                    &client.ClientComms{},
		tryReconnect:             make(chan struct{}),
		TLS:                      true,
		ndf:                      ndf,
		ReceptionGatewayIndex:    len(ndf.Gateways) - 1,
		TransmissionGatewayIndex: 0,
		connectionStatusCallback: callback,
		connectionStatus:         &status,
	}

	return &cm
}

// Connects to gateways using TLS filepaths to create credential information
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
			if gateway.TlsCertificate != "" && cm.TLS {
				gwCreds = []byte(gateway.TlsCertificate)
			}
			gwID := id.NewNodeFromBytes(cm.ndf.Nodes[i].ID).NewGateway()
			globals.Log.INFO.Printf("Connecting to gateway %s at %s...",
				gwID.String(), gateway.Address)
			err = cm.Comms.ConnectToRemote(gwID, gateway.Address,
				gwCreds, false)

			if err != nil {
				errChan <- errors.New(fmt.Sprintf(
					"Failed to connect to gateway %s at %s: %+v",
					gwID.String(), gateway.Address, err))
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
	cm.RegistrationVersion = registrationVersion.Version
	return nil
}

func (cm *CommManager) GetConnectionCallback() ConnectionStatusCallback {
	return cm.connectionStatusCallback
}

//GetUpdatedNDF: Connects to the permissioning server to get the updated NDF from it
func (cm *CommManager) GetUpdatedNDF() (*ndf.NetworkDefinition, error) {
	connected, err := cm.ConnectToPermissioning()
	cm.lock.Lock()
	cm.lock.RLock()
	defer cm.DisconnectFromPermissioning()
	defer cm.lock.Unlock()
	defer cm.lock.RUnlock()

	if err != nil {
		cm.ndf = &ndf.NetworkDefinition{}
		return &ndf.NetworkDefinition{}, err
	}

	if !connected {
		errMsg := fmt.Sprintf("Failed to connect to permissioning server")
		globals.Log.ERROR.Printf(errMsg)
		return &ndf.NetworkDefinition{}, errors.New(errMsg)
	}

	//Hash the client's ndf for comparison with registration's ndf
	hash := sha256.New()
	ndfBytes := cm.ndf.Serialize()
	hash.Write(ndfBytes)
	ndfHash := hash.Sum(nil)

	//Put the hash in a message
	msg := &mixmessages.NDFHash{Hash: ndfHash}

	//Send the hash to registration
	response, err := cm.Comms.SendGetUpdatedNDF(ConnAddr(PermissioningAddrID), msg)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to get ndf from permissioning: %v", err)
		return &ndf.NetworkDefinition{}, errors.New(errMsg)
	}

	//Response should not be nil, check comms
	if response == nil {
		globals.Log.ERROR.Printf("Response given was an unexpected nil, check comms")
		return cm.ndf, nil
	}

	//If there was no error and the response is an empty byte slice, client's ndf is up-to-date
	if bytes.Compare(response.Ndf, make([]byte, 0)) == 0 {
		globals.Log.DEBUG.Printf("Client NDF up-to-date")
		return cm.ndf, nil
	}

	//FixMe: use verify instead? Probs need to add a signature to ndf, like in registration's getupdate?
	//Otherwise pull the ndf out of the response
	updatedNdf, _, err := ndf.DecodeNDF(string(response.Ndf))
	if err != nil {
		//If there was an error decoding ndf
		errMsg := fmt.Sprintf("Failed to decode response to ndf: %v", err)
		return &ndf.NetworkDefinition{}, errors.New(errMsg)
	}

	//Set the updated ndf to be the client's ndf
	cm.ndf = updatedNdf
	//Update the amount of gateways
	cm.ReceptionGatewayIndex = len(updatedNdf.Gateways) - 1

	return updatedNdf, nil
}

// Utility method, returns whether the local version and remote version are
// compatible
func (cm *CommManager) CheckVersion() (bool, error) {
	return checkVersion(globals.SEMVER, cm.RegistrationVersion)
}

// There's currently no need to keep connected to permissioning constantly,
// so we have functions to connect to and disconnect from it when a connection
// to permissioning is needed
func (cm *CommManager) ConnectToPermissioning() (connected bool, err error) {
	if cm.ndf.Registration.Address != "" {
		var regCert []byte
		if cm.ndf.Registration.TlsCertificate != "" && cm.TLS {
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
