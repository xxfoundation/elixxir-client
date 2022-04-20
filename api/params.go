///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package api

import (
	"gitlab.com/elixxir/client/auth"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	"gitlab.com/elixxir/client/e2e/rekey"
)

type Params struct {
	E2E     e2e.Params
	CMix    cmix.Params
	Network cmix.CMIXParams
	Session session.Params
	Auth    auth.Param
	Rekey   rekey.Params
}

func GetDefaultParams() Params {
	return Params{
		E2E:     e2e.GetDefaultParams(),
		CMix:    cmix.GetDefaultParams(),
		Network: cmix.GetDefaultCMIXParams(),
		Session: session.GetDefaultParams(),
		Auth:    auth.GetDefaultParams(),
		Rekey:   rekey.GetDefaultParams(),
	}
}
