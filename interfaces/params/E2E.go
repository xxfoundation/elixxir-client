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
	Type                 SendType
	RetryCount           int
	OnlyNotifyOnLastSend bool
	CMIX
}

func GetDefaultE2E() E2E {
	return E2E{
		Type:                 Standard,
		CMIX:                 GetDefaultCMIX(),
		OnlyNotifyOnLastSend: true,
		RetryCount:           10,
	}
}
func (e E2E) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

// Obtain default E2E parameters, or override with given parameters if set
func GetE2EParameters(params string) (E2E, error) {
	p := GetDefaultE2E()
	if len(params) > 0 {
		err := json.Unmarshal([]byte(params), &p)
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

// Network E2E Params

type E2ESessionParams struct {
	// using the DH as a seed, both sides generate a number
	// of keys to use before they must rekey because
	// there are no keys to use.
	MinKeys uint16
	MaxKeys uint16
	// the percent of keys before a rekey is attempted. must be <0
	RekeyThreshold float64
	// extra keys generated and reserved for rekey attempts. This
	// many keys are not allowed to be used for sending messages
	// in order to ensure there are extras for rekeying.
	NumRekeys uint16
}

// DEFAULT KEY GENERATION PARAMETERS
// Hardcoded limits for keys
// sets the number of keys very high, but with a low rekey threshold. In this case, if the other party is online, you will read
const (
	minKeys       uint16  = 1000
	maxKeys       uint16  = 2000
	rekeyThrshold float64 = 0.05
	numReKeys     uint16  = 16
)

func GetDefaultE2ESessionParams() E2ESessionParams {
	return E2ESessionParams{
		MinKeys:        minKeys,
		MaxKeys:        maxKeys,
		RekeyThreshold: rekeyThrshold,
		NumRekeys:      numReKeys,
	}
}

func (p E2ESessionParams) String() string {
	return fmt.Sprintf("Params{ MinKeys: %d, MaxKeys: %d, NumRekeys: %d }",
		p.MinKeys, p.MaxKeys, p.NumRekeys)
}
