///////////////////////////////////////////////////////////////////////////////
// Copyright © 2021 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// Contains params-related bindings

package old

import (
	"gitlab.com/elixxir/client/interfaces/params"
)

func GetCMIXParams() (string, error) {
	p, err := params.GetDefaultCMIX().Marshal()
	return string(p), err
}

func GetE2EParams() (string, error) {
	p, err := params.GetDefaultE2E().Marshal()
	return string(p), err
}

func GetNetworkParams() (string, error) {
	p, err := params.GetDefaultNetwork().Marshal()
	return string(p), err
}

func GetUnsafeParams() (string, error) {
	p, err := params.GetDefaultUnsafe().Marshal()
	return string(p), err
}
