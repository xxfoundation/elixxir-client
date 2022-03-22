///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package bindings

// NodeRegistrationsStatus structure for returning nodes registration statuses
// for bindings.
type NodeRegistrationsStatus struct {
	registered int
	total      int
}

// GetRegistered returns the number of nodes registered with the client.
func (nrs *NodeRegistrationsStatus) GetRegistered() int {
	return nrs.registered
}

// GetTotal return the total of nodes currently in the network.
func (nrs *NodeRegistrationsStatus) GetTotal() int {
	return nrs.total
}
