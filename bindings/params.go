///////////////////////////////////////////////////////////////////////////////
// Copyright © 2021 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// Contains params-related bindings

package bindings

import (
	"gitlab.com/elixxir/client/interfaces/params"
)

func (c *Client) GetCMIXParams() (string, error) {
	p, err := params.GetDefaultCMIX().Marshal()
	return string(p), err
}

func (c *Client) GetE2EParams() (string, error) {
	p, err := params.GetDefaultE2E().Marshal()
	return string(p), err
}

func (c *Client) GetNetworkParams() (string, error) {
	p, err := params.GetDefaultNetwork().Marshal()
	return string(p), err
}

func (c *Client) GetUnsafeParams() (string, error) {
	p, err := params.GetDefaultUnsafe().Marshal()
	return string(p), err
}
