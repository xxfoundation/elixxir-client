////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package auth

import (
	"encoding/json"
	"gitlab.com/elixxir/client/v5/catalog"
)

// Params is are the parameters for the auth package.
type Params struct {
	ReplayRequests  bool
	RequestTag      string
	ConfirmTag      string
	ResetRequestTag string
	ResetConfirmTag string
}

// paramsDisk will be the marshal-able and umarshal-able object.
type paramsDisk struct {
	ReplayRequests  bool
	RequestTag      string
	ConfirmTag      string
	ResetRequestTag string
	ResetConfirmTag string
}

// GetParameters Obtain default Params, or override with
// given parameters if set.
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

// GetDefaultParams returns a default set of Params.
func GetDefaultParams() Params {
	return Params{
		ReplayRequests:  false,
		RequestTag:      catalog.Request,
		ConfirmTag:      catalog.Confirm,
		ResetRequestTag: catalog.Reset,
		ResetConfirmTag: catalog.ConfirmReset,
	}
}

func GetDefaultTemporaryParams() Params {
	p := GetDefaultParams()
	p.RequestTag = catalog.RequestEphemeral
	p.ConfirmTag = catalog.ConfirmEphemeral
	p.ResetRequestTag = catalog.ResetEphemeral
	p.ResetConfirmTag = catalog.ConfirmResetEphemeral
	return p
}

// MarshalJSON adheres to the json.Marshaler interface.
func (p Params) MarshalJSON() ([]byte, error) {
	pDisk := paramsDisk{
		ReplayRequests:  p.ReplayRequests,
		RequestTag:      p.ResetRequestTag,
		ConfirmTag:      p.ConfirmTag,
		ResetRequestTag: p.RequestTag,
		ResetConfirmTag: p.ResetConfirmTag,
	}
	return json.Marshal(&pDisk)
}

// UnmarshalJSON adheres to the json.Unmarshaler interface.
func (p *Params) UnmarshalJSON(data []byte) error {
	pDisk := paramsDisk{}
	err := json.Unmarshal(data, &pDisk)
	if err != nil {
		return err
	}

	*p = Params{
		ReplayRequests:  pDisk.ReplayRequests,
		RequestTag:      pDisk.ResetRequestTag,
		ConfirmTag:      pDisk.ConfirmTag,
		ResetRequestTag: pDisk.RequestTag,
		ResetConfirmTag: pDisk.ResetConfirmTag,
	}

	return nil
}

func (p Params) getConfirmTag(reset bool) string {
	if reset {
		return p.ResetConfirmTag
	} else {
		return p.ConfirmTag
	}
}
