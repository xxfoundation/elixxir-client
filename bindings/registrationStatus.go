///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package bindings

// NodeRegistrationsStatus structure for returning node registration statuses
// for bindings.
type NodeRegistrationsStatus struct {
	registered int
	inProgress int
}

// GetRegistered returns the number of nodes registered with the client.
func (nrs *NodeRegistrationsStatus) GetRegistered() int {
	return nrs.registered
}

// GetInProgress return the number of nodes currently registering.
func (nrs *NodeRegistrationsStatus) GetInProgress() int {
	return nrs.inProgress
}
