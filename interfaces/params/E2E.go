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
	"gitlab.com/elixxir/crypto/e2e"
)

type E2E struct {
	Type       SendType
	RetryCount int
	OnlyNotifyOnLastSend bool
	CMIX
}

func GetDefaultE2E() E2E {
	return E2E{
		Type:       Standard,
		CMIX:       GetDefaultCMIX(),
		OnlyNotifyOnLastSend: true,
		RetryCount: 10,
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

// DEFAULT KEY GENERATION PARAMETERS
// Hardcoded limits for keys
// With 16 receiving states we can hold
// 16*64=1024 dirty bits for receiving keys
// With that limit, and setting maxKeys to 800,
// we need a Threshold of 224, and a scalar
// smaller than 1.28 to ensure we never generate
// more than 1024 keys
// With 1 receiving states for ReKeys we can hold
// 64 Rekeys
const (
	minKeys   uint16  = 500
	maxKeys   uint16  = 800
	ttlScalar float64 = 1.2 // generate 20% extra keys
	threshold uint16  = 224
	numReKeys uint16  = 16
)

type E2ESessionParams struct {
	MinKeys   uint16
	MaxKeys   uint16
	NumRekeys uint16
	e2e.TTLParams
}

func GetDefaultE2ESessionParams() E2ESessionParams {
	return E2ESessionParams{
		MinKeys:   minKeys,
		MaxKeys:   maxKeys,
		NumRekeys: numReKeys,
	}
}

func (p E2ESessionParams) String() string {
	return fmt.Sprintf("Params{ MinKeys: %d, MaxKeys: %d, NumRekeys: %d }",
		p.MinKeys, p.MaxKeys, p.NumRekeys)
}
