///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package api

import (
	"encoding/json"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/e2e/ratchet/partner/session"
)

type Params struct {
	CMix    cmix.Params
	Session session.Params
}

func GetDefaultParams() Params {
	return Params{
		CMix:    cmix.GetDefaultParams(),
		Session: session.GetDefaultParams(),
	}
}

// GetParameters returns the default Params, or override with given
// parameters, if set.
func GetParameters(params string) (Params, error) {
	p := GetDefaultParams()
	if len(params) > 0 {
		err := json.Unmarshal([]byte(params), &p)
		if err != nil {
			return Params{}, err
		}
	}
	return p, nil
}
