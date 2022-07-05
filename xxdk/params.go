///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package xxdk

// params.go define the high level parameters structures (which embed E2E and
// CMIX params respectively) that are passed down into the core xxdk modules.

import (
	"encoding/json"

	"gitlab.com/elixxir/client/auth"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	"gitlab.com/elixxir/client/e2e/rekey"
)

// CMIXParams contains the parameters for Network tracking and for
// specific CMIX messaging settings.
//
// FIXME: this breakdown could be cleaner and is an unfortunate side effect of
//        several refactors of the codebase.
type CMIXParams struct {
	Network cmix.Params
	CMIX    cmix.CMIXParams
}

// E2EParams holds all the settings for e2e and it's various submodules.
//
// NOTE: "Base" wraps cmix.CMIXParams to control message send params,
//       so xxdk library users should copy the desired settings to both.
// FIXME: this should not wrap a copy of cmix.CMIXParams.
type E2EParams struct {
	Session        session.Params
	Base           e2e.Params
	Rekey          rekey.Params
	EphemeralRekey rekey.Params
	Auth           auth.Params
}

////////////////////////////////////////
// -- CMix Params Helper Functions -- //
////////////////////////////////////////

func GetDefaultCMixParams() CMIXParams {
	return CMIXParams{
		Network: cmix.GetDefaultParams(),
		CMIX:    cmix.GetDefaultCMIXParams(),
	}
}

// Unmarshal fills an empty object with the deserialized contents of jsonData
func (p *CMIXParams) Unmarshal(jsonData []byte) error {
	return json.Unmarshal(jsonData, p)
}

// Marshal creates json data of the object
func (p *CMIXParams) Marshal() ([]byte, error) {
	return json.Marshal(p)
}

////////////////////////////////////////
// -- E2E Params Helper Functions --  //
////////////////////////////////////////

func GetDefaultE2EParams() E2EParams {
	return E2EParams{
		Session:        session.GetDefaultParams(),
		Base:           e2e.GetDefaultParams(),
		Rekey:          rekey.GetDefaultParams(),
		EphemeralRekey: rekey.GetDefaultEphemeralParams(),
		Auth:           auth.GetDefaultParams(),
	}
}

// Unmarshal fills an empty object with the deserialized contents of jsonData
func (p *E2EParams) Unmarshal(jsonData []byte) error {
	return json.Unmarshal(jsonData, p)
}

// Marshal creates json data of the object
func (p *E2EParams) Marshal() ([]byte, error) {
	return json.Marshal(p)
}
