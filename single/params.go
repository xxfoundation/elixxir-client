////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package single

import (
	"gitlab.com/elixxir/client/cmix"
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

// GetDefaultRequestParams returns a RequestParams with the default
// configuration.
func GetDefaultRequestParams() RequestParams {
	return RequestParams{
		Timeout:             defaultRequestTimeout,
		MaxResponseMessages: defaultMaxResponseMessages,
		CmixParams:          cmix.GetDefaultCMIXParams(),
	}
}
