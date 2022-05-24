///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package api

import (
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
