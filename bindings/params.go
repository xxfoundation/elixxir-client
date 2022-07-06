///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// params.go provides functions for getting and setting parameters in bindings.

package bindings

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/fileTransfer"
	"gitlab.com/elixxir/client/single"
	"gitlab.com/elixxir/client/xxdk"
)

// GetDefaultCMixParams returns a JSON serialized object with all of the
// CMIX parameters and their default values. Call this function and modify
// the json to change CMIX settings.
func GetDefaultCMixParams() []byte {
	defaultParams := xxdk.GetDefaultCMixParams()
	data, err := defaultParams.Marshal()
	if err != nil {
		jww.FATAL.Panicf("Unexpected error: %+v", err)
	}
	return data
}

// GetDefaultE2EParams returns a JSON serialized object with all of the
// E2E parameters and their default values. Call this function and modify
// the json to change E2E settings.
func GetDefaultE2EParams() []byte {
	defaultParams := xxdk.GetDefaultE2EParams()
	data, err := defaultParams.Marshal()
	if err != nil {
		jww.FATAL.Panicf("Unexpected error: %+v", err)
	}
	return data
}

func GetDefaultFileTransferParams() []byte {
	defaultParams := fileTransfer.DefaultParams()
	data, err := defaultParams.MarshalJSON()
	if err != nil {
		jww.FATAL.Panicf("Unexpected error: %+v", err)
	}
	return data
}

func GetDefaultSingleUseParams() []byte {
	defaultParams := single.GetDefaultRequestParams()
	data, err := defaultParams.MarshalJSON()
	if err != nil {
		jww.FATAL.Panicf("Unexpected error: %+v", err)
	}
	return data
}

func parseSingleUseParams(data []byte) (single.RequestParams, error) {
	p := &single.RequestParams{}
	return *p, p.UnmarshalJSON(data)
}

func parseFileTransferParams(data []byte) (fileTransfer.Params, error) {
	p := &fileTransfer.Params{}
	return *p, p.UnmarshalJSON(data)
}

func parseCMixParams(data []byte) (xxdk.CMIXParams, error) {
	p := &xxdk.CMIXParams{}
	err := p.Unmarshal(data)
	return *p, err
}

func parseE2EParams(data []byte) (xxdk.E2EParams, error) {
	p := &xxdk.E2EParams{}
	err := p.Unmarshal(data)
	return *p, err
}
