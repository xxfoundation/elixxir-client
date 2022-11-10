////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package nodes

import (
	"gitlab.com/elixxir/client/v5/stoppable"
	"gitlab.com/elixxir/client/v5/storage/versioned"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/network"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

// Registrar is an interface for managing the registrations
// for cMix nodes.
type Registrar interface {
	// StartProcesses initiates numParallel amount of threads
	// to register with nodes.
	StartProcesses(numParallel uint) stoppable.Stoppable

	//PauseNodeRegistrations stops all node registrations
	//and returns a function to resume them
	PauseNodeRegistrations(timeout time.Duration) error

	// ChangeNumberOfNodeRegistrations changes the number of parallel node
	// registrations up to the initialized maximum
	ChangeNumberOfNodeRegistrations(toRun int, timeout time.Duration) error

	// GetNodeKeys returns a MixCypher for the topology and a list of nodes it did
	// not have a key for. If there are missing keys, then returns nil.
	GetNodeKeys(topology *connect.Circuit) (MixCypher, error)

	// HasNode returns whether Registrar has registered with this cMix node
	HasNode(nid *id.ID) bool

	// RemoveNode removes the node from the registrar
	RemoveNode(nid *id.ID)

	// NumRegisteredNodes returns the number of registered nodes.
	NumRegisteredNodes() int

	// GetInputChannel returns the send-only channel for registering with
	// a cMix node.
	GetInputChannel() chan<- network.NodeGateway

	// TriggerNodeRegistration initiates a registration with the given
	// cMix node by sending on the registrar's registration channel.
	TriggerNodeRegistration(nid *id.ID)
}

// MixCypher is an interface for the cryptographic operations done in order
// to encrypt a cMix message to a node.
type MixCypher interface {
	// Encrypt encrypts the given message for cMix. Panics if the passed
	// message is not sized correctly for the group.
	Encrypt(msg format.Message, salt []byte, roundID id.Round) (
		format.Message, [][]byte)

	// MakeClientGatewayAuthMAC generates the MAC the gateway will
	// check when receiving a cMix message.
	MakeClientGatewayAuthMAC(salt, digest []byte) []byte
}

// RegisterNodeCommsInterface is a sub-interface of client.Comms containing
// the send function for registering with a cMix node.
type RegisterNodeCommsInterface interface {
	SendRequestClientKeyMessage(host *connect.Host,
		message *pb.SignedClientKeyRequest) (*pb.SignedKeyResponse, error)
}

// session is a sub-interface of the storage.Session interface relevant to
// the methods used in this package.
type session interface {
	GetTransmissionID() *id.ID
	IsPrecanned() bool
	GetCmixGroup() *cyclic.Group
	GetKV() *versioned.KV
	GetTransmissionRSA() *rsa.PrivateKey
	GetRegistrationTimestamp() time.Time
	GetTransmissionSalt() []byte
	GetTransmissionRegistrationValidationSignature() []byte
}
