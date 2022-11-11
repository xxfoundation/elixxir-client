////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package single

import (
	"encoding/json"
	"gitlab.com/elixxir/client/v4/cmix"
	"time"
)

// Default values.
const (
	defaultRequestTimeout      = 30 * time.Second
	defaultMaxResponseMessages = 255
)

// RequestParams contains configurable parameters for sending a single-use
// request message.
type RequestParams struct {
	// Timeout is the duration to wait before timing out while sending a request
	Timeout time.Duration

	// MaxResponseMessages is the maximum number of messages allowed in the
	// response to the request
	MaxResponseMessages uint8

	// CmixParams is the parameters used in sending a cMix message
	CmixParams cmix.CMIXParams
}

// requestParamsDisk will be the marshal-able and umarshal-able object.
type requestParamsDisk struct {
	Timeout             time.Duration
	MaxResponseMessages uint8
	CmixParams          cmix.CMIXParams
}

// GetDefaultRequestParams returns a RequestParams with the default
// configuration.
func GetDefaultRequestParams() RequestParams {
	return RequestParams{
		Timeout:             defaultRequestTimeout,
		MaxResponseMessages: defaultMaxResponseMessages,
		CmixParams:          cmix.GetDefaultCMIXParams(),
	}
}

// GetParameters returns the default network parameters, or override with given
// parameters, if set.
func GetParameters(params string) (RequestParams, error) {
	p := GetDefaultRequestParams()
	if len(params) > 0 {
		err := json.Unmarshal([]byte(params), &p)
		if err != nil {
			return RequestParams{}, err
		}
	}
	return p, nil
}

// MarshalJSON adheres to the json.Marshaler interface.
func (rp RequestParams) MarshalJSON() ([]byte, error) {
	pDisk := requestParamsDisk{
		Timeout:             rp.Timeout,
		MaxResponseMessages: rp.MaxResponseMessages,
		CmixParams:          rp.CmixParams,
	}

	return json.Marshal(&pDisk)

}

// UnmarshalJSON adheres to the json.Unmarshaler interface.
func (rp *RequestParams) UnmarshalJSON(data []byte) error {
	pDisk := requestParamsDisk{}
	err := json.Unmarshal(data, &pDisk)
	if err != nil {
		return err
	}

	*rp = RequestParams{
		Timeout:             pDisk.Timeout,
		MaxResponseMessages: pDisk.MaxResponseMessages,
		CmixParams:          pDisk.CmixParams,
	}

	return nil
}
