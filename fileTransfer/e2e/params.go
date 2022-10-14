////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package e2e

import "encoding/json"

const (
	defaultNotifyUponCompletion = true
)

// Params contains parameters used for E2E file transfer.
type Params struct {
	// NotifyUponCompletion indicates if a final notification message is sent
	// to the recipient on completion of file transfer. If true, the ping is
	// sent.
	NotifyUponCompletion bool
}

// DefaultParams returns a Params object filled with the default values.
func DefaultParams() Params {
	return Params{
		NotifyUponCompletion: defaultNotifyUponCompletion,
	}
}

// GetParameters returns the default network parameters, or override with given
// parameters, if set. Returns an error if provided invalid JSON. If the JSON is
// valid but does not match the Params structure, the default parameters will be
// returned.
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
