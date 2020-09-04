////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"gitlab.com/elixxir/client/api"
)

// NewClient connects and registers to the network using a json encoded
// network information string and then creates a new client at the specified
// storageDir using the specified password. This function will fail
// when:
//   - network information cannot be read or the client cannot connect
//     to the network and register within the defined timeout.
//   - storageDir does not exist and cannot be created
//   - It cannot create, read, or write files inside storageDir
//   - Client files already exist inside storageDir.
//   - cryptographic functionality is unavailable (e.g. random number
//     generation)
// The password is passed as a byte array so that it can be cleared from
// memory and stored as securely as possible using the memguard library.
// NewClient will block until the client has completed registration with
// the network permissioning server.
//
// Users of this function should delete the storage directory on error.
func NewClient(network, storageDir string, password []byte) (Client, error) {
	// TODO: This should wrap the bindings ClientImpl, when available.
	return api.NewClient(network, storageDir, password)
}

// LoadClient will load an existing client from the storageDir
// using the password. This will fail if the client doesn't exist or
// the password is incorrect.
// The password is passed as a byte array so that it can be cleared from
// memory and stored as securely as possible using the memguard library.
// LoadClient does not block on network connection, and instead loads and
// starts subprocesses to perform network operations.
func LoadClient(storageDir string, password []byte) (Client, error) {
	// TODO: This should wrap the bindings ClientImpl, when available.
	return api.LoadClient(storageDir, password)
}
