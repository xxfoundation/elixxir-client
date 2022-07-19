///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/xxdk"
)

// init sets the log level
func init() {
	jww.SetLogThreshold(jww.LevelInfo)
	jww.SetStdoutThreshold(jww.LevelInfo)
}

// cmixTrackerSingleton is used to track Cmix objects so that
// they can be referenced by id back over the bindings
var cmixTrackerSingleton = &cmixTracker{
	clients: make(map[int]*Cmix),
	count:   0,
}

// Cmix wraps the xxdk.Cmix struct, implementing additional functions
// to support the gomobile Cmix interface
type Cmix struct {
	api *xxdk.Cmix
	id  int
}

// NewCmix creates client storage, generates keys, connects, and registers
// with the network. Note that this does not register a username/identity, but
// merely creates a new cryptographic identity for adding such information
// at a later date.
//
// Users of this function should delete the storage directory on error.
func NewCmix(ndfJSON, storageDir string, password []byte, registrationCode string) error {
	if err := xxdk.NewCmix(ndfJSON, storageDir, password, registrationCode); err != nil {
		return errors.New(fmt.Sprintf("Failed to create new client: %+v",
			err))
	}
	return nil
}

// LoadCmix will load an existing client from the storageDir
// using the password. This will fail if the client doesn't exist or
// the password is incorrect.
// The password is passed as a byte array so that it can be cleared from
// memory and stored as securely as possible using the memguard library.
// LoadCmix does not block on network connection, and instead loads and
// starts subprocesses to perform network operations.
// TODO: add in custom parameters instead of the default
func LoadCmix(storageDir string, password []byte, cmixParamsJSON []byte) (*Cmix,
	error) {
	if len(cmixParamsJSON) == 0 {
		jww.WARN.Printf("cmix params not specified, using defaults...")
		cmixParamsJSON = GetDefaultCMixParams()
	}

	params, err := parseCMixParams(cmixParamsJSON)
	if err != nil {
		return nil, err
	}

	client, err := xxdk.LoadCmix(storageDir, password, params)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("LoadCmix failed: %+v", err))
	}

	return cmixTrackerSingleton.make(client), nil
}

func (c *Cmix) GetID() int {
	return c.id
}

// cmixTracker is a singleton used to keep track of extant Cmix objects,
// preventing race conditions created by passing it over the bindings
type cmixTracker struct {
	clients map[int]*Cmix
	count   int
	mux     sync.RWMutex
}

// make a Cmix from an xxdk.Cmix, assigns it a unique ID,
// and adds it to the cmixTracker
func (ct *cmixTracker) make(c *xxdk.Cmix) *Cmix {
	ct.mux.Lock()
	defer ct.mux.Unlock()

	id := ct.count
	ct.count++

	ct.clients[id] = &Cmix{
		api: c,
		id:  id,
	}

	return ct.clients[id]
}

// get a Cmix from the cmixTracker given its ID
func (ct *cmixTracker) get(id int) (*Cmix, error) {
	ct.mux.RLock()
	defer ct.mux.RUnlock()

	c, exist := ct.clients[id]
	if !exist {
		return nil, errors.Errorf("Cannot get client for id %d, client "+
			"does not exist", id)
	}

	return c, nil
}

// delete a Cmix if it exists in the cmixTracker
func (ct *cmixTracker) delete(id int) {
	ct.mux.Lock()
	defer ct.mux.Unlock()

	delete(ct.clients, id)
}
