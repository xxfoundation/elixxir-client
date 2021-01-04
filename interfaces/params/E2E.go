///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package params

import (
	"encoding/json"
	"fmt"
)

type E2E struct {
	Type SendType
	CMIX
}

func GetDefaultE2E() E2E {
	return E2E{Type: Standard,
		CMIX: GetDefaultCMIX(),
	}
}

func (e *E2E) MarshalJSON() ([]byte, error) {
	return json.Marshal(e)
}

func (e *E2E) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, e)
}

// Obtain default E2E parameters, or override with given parameters if set
func GetE2EParameters(params string) (E2E, error) {
	p := GetDefaultE2E()
	if len(params) > 0 {
		err := p.UnmarshalJSON([]byte(params))
		if err != nil {
			return E2E{}, err
		}
	}
	return p, nil
}

type SendType uint8

const (
	Standard    SendType = 0
	KeyExchange SendType = 1
)

func (st SendType) String() string {
	switch st {
	case Standard:
		return "Standard"
	case KeyExchange:
		return "KeyExchange"
	default:
		return fmt.Sprintf("Unknown SendType %v", uint8(st))
	}
}
