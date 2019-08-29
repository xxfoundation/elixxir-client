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
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/comms/client"
	"sync/atomic"

	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/ndf"
	"sync"
	"time"
)

const maxAttempts = 15
const maxBackoffTime = 180
const PermissioningAddrID = "registration"

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

// Connects to gateways and registration server (if needed)
// using TLS filepaths to create credential information
// for connection establishment
func (cm *CommManager) Connect() error {
	var err error
	if len(cm.ndf.Gateways) < 1 {
		return errors.New("could not connect due to invalid number of nodes")
	}

	cm.setConnectionStatus(Connecting, 0)

	cm.Comms.ConnectionManager.SetMaxRetries(1)

	//connect to the registration server
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
			return errors.New(fmt.Sprintf(
				"Failed connecting to permissioning: %+v", err))
		}
		globals.Log.INFO.Printf(
			"Connected to permissioning at %v successfully!",
			cm.ndf.Registration.Address)

		// At this point, we should be connected to the registration server
		// So, we'll make sure that our client version is compatible with the
		// network
		registrationConnID := ConnAddr(PermissioningAddrID)
		registrationVersion, err := cm.Comms.SendGetCurrentClientVersionMessage(registrationConnID)
		if err != nil {
			return errors.Wrap(err, "Couldn't get current version from permissioning")
		}
		versionOk, err := CheckVersion(SEMVER, registrationVersion.Version)
		if err != nil {
			return err
		}
		if !versionOk {
			return errors.New(fmt.Sprintf(
				"Please update local client library.\n"+
					"Local client version: %v, minimum version: %v",
				SEMVER, registrationVersion.Version))
		}
	} else {
		globals.Log.WARN.Printf("Unable to find permissioning server!")
	}

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

func (cm *CommManager) Disconnect() {
	cm.Comms.DisconnectAll()
}

func (cm *CommManager) GetConnectionStatus() uint32 {
	return atomic.LoadUint32(cm.connectionStatus)
}

func (cm *CommManager) setConnectionStatus(status uint32, timeout int) {
	atomic.SwapUint32(cm.connectionStatus, status)
	cm.connectionStatusCallback(status, timeout)
}

func toSeconds(duration time.Duration) int {
	return int(duration) / int(time.Second)
}
