///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package params

import "encoding/json"

type Unsafe struct {
	CMIX
}

func GetDefaultUnsafe() Unsafe {
	return Unsafe{CMIX: GetDefaultCMIX()}
}

func (u *Unsafe) MarshalJSON() ([]byte, error) {
	return json.Marshal(u)
}

func (u *Unsafe) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, u)
}

// Obtain default Unsafe parameters, or override with given parameters if set
func GetUnsafeParameters(params string) (Unsafe, error) {
	p := GetDefaultUnsafe()
	if len(params) > 0 {
		err := p.UnmarshalJSON([]byte(params))
		if err != nil {
			return Unsafe{}, err
		}
	}
	return p, nil
}
