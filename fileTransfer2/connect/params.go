////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package connect

import (
	"encoding/json"
)

const (
	defaultNotifyUponCompletion = true
)

// Params contains parameters used for Connection file transfer.
type Params struct {
	// NotifyUponCompletion indicates if a final notification message is sent
	// to the recipient on completion of file transfer. If true, the ping is
	NotifyUponCompletion bool
}

// paramsDisk will be the marshal-able and umarshal-able object.
type paramsDisk struct {
	NotifyUponCompletion bool
}

// DefaultParams returns a Params object filled with the default values.
func DefaultParams() Params {
	return Params{
		NotifyUponCompletion: defaultNotifyUponCompletion,
	}
}

// GetParameters returns the default Params, or override with given
// parameters, if set.
func GetParameters(params string) (Params, error) {
	p := DefaultParams()
	if len(params) > 0 {
		err := json.Unmarshal([]byte(params), &p)
		if err != nil {
			return Params{}, err
		}
	}
	return p, nil
}

// MarshalJSON adheres to the json.Marshaler interface.
func (r Params) MarshalJSON() ([]byte, error) {
	pDisk := paramsDisk{NotifyUponCompletion: r.NotifyUponCompletion}
	return json.Marshal(&pDisk)

}

// UnmarshalJSON adheres to the json.Unmarshaler interface.
func (r *Params) UnmarshalJSON(data []byte) error {
	pDisk := paramsDisk{}
	err := json.Unmarshal(data, &pDisk)
	if err != nil {
		return err
	}

	*r = Params{NotifyUponCompletion: pDisk.NotifyUponCompletion}

	return nil
}
