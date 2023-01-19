////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package e2e

import (
	"encoding/json"
	"time"

	"gitlab.com/elixxir/client/v4/catalog"
	"gitlab.com/elixxir/client/v4/cmix"
)

type Params struct {
	// Tag to use to generate the service.
	ServiceTag string
	// Often, for notifications purposes, all messages except the
	// last should use a silent service. This allows a
	LastServiceTag string

	// The parameters adjust how the code behaves if there are not
	// keys available.  the number of times the code will attempt
	// to get a key to encrypt with
	KeyGetRetryCount uint
	// Delay between attempting to get kets
	KeyGeRetryDelay time.Duration

	//Authorizes the message to use a key reserved for rekeying. Do not use
	//unless sending a rekey
	Rekey bool

	cmix.CMIXParams
}

// paramsDisk will be the marshal-able and umarshal-able object.
type paramsDisk struct {
	ServiceTag       string
	LastServiceTag   string
	KeyGetRetryCount uint
	KeyGeRetryDelay  time.Duration
	Rekey            bool
	cmix.CMIXParams
}

// GetDefaultParams returns a default set of Params.
func GetDefaultParams() Params {
	return Params{
		ServiceTag:     catalog.Silent,
		LastServiceTag: catalog.E2e,

		KeyGetRetryCount: 10,
		KeyGeRetryDelay:  500 * time.Millisecond,

		Rekey:      false,
		CMIXParams: cmix.GetDefaultCMIXParams(),
	}
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

// MarshalJSON adheres to the json.Marshaler interface.
func (p Params) MarshalJSON() ([]byte, error) {
	pDisk := paramsDisk{
		ServiceTag:       p.ServiceTag,
		LastServiceTag:   p.LastServiceTag,
		KeyGetRetryCount: p.KeyGetRetryCount,
		KeyGeRetryDelay:  p.KeyGeRetryDelay,
		Rekey:            p.Rekey,
		CMIXParams:       p.CMIXParams,
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
		ServiceTag:       pDisk.ServiceTag,
		LastServiceTag:   pDisk.LastServiceTag,
		KeyGetRetryCount: pDisk.KeyGetRetryCount,
		KeyGeRetryDelay:  pDisk.KeyGeRetryDelay,
		Rekey:            pDisk.Rekey,
		CMIXParams:       pDisk.CMIXParams,
	}

	return nil
}

// String implements stringer interface by returning a json string
func (p *Params) String() string {
	json, _ := p.MarshalJSON()
	return string(json)
}
