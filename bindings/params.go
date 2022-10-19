////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

// params.go provides functions for getting and setting parameters in bindings.

package bindings

import (
	jww "github.com/spf13/jwalterweatherman"
	xxdk "gitlab.com/elixxir/client/xxdk2"

	//"gitlab.com/elixxir/client/fileTransfer"
	//e2eFileTransfer "gitlab.com/elixxir/client/fileTransfer/e2e"
	"gitlab.com/elixxir/client/single"
	//"gitlab.com/elixxir/client/xxdk"
)

// GetDefaultCMixParams returns a JSON serialized object with all of the cMix
// parameters and their default values. Call this function and modify the JSON
// to change cMix settings.
func GetDefaultCMixParams() []byte {
	defaultParams := xxdk.GetDefaultCMixParams()
	data, err := defaultParams.Marshal()
	if err != nil {
		jww.FATAL.Panicf("Failed to JSON marshal cMix params: %+v", err)
	}
	return data
}

// GetDefaultE2EParams returns a JSON serialized object with all of the E2E
// parameters and their default values. Call this function and modify the JSON
// to change E2E settings.
func GetDefaultE2EParams() []byte {
	defaultParams := xxdk.GetDefaultE2EParams()
	data, err := defaultParams.Marshal()
	if err != nil {
		jww.FATAL.Panicf("Failed to JSON marshal E2E params: %+v", err)
	}
	return data
}

// GetDefaultSingleUseParams returns a JSON serialized object with all the
// single-use parameters and their default values. Call this function and modify
// the JSON to change single use settings.
func GetDefaultSingleUseParams() []byte {
	defaultParams := single.GetDefaultRequestParams()
	data, err := defaultParams.MarshalJSON()
	if err != nil {
		jww.FATAL.Panicf("Failed to JSON marshal single-use params: %+v", err)
	}
	return data
}

// parseSingleUseParams is a helper function which parses a JSON marshalled
// [single.RequestParams].
func parseSingleUseParams(data []byte) (single.RequestParams, error) {
	p := &single.RequestParams{}
	return *p, p.UnmarshalJSON(data)
}

// parseCMixParams is a helper function which parses a JSON marshalled
// [xxdk.CMIXParams].
func parseCMixParams(data []byte) (xxdk.CMIXParams, error) {
	if len(data) == 0 {
		jww.WARN.Printf("cMix params not specified, using defaults...")
		data = GetDefaultCMixParams()
	}

	p := &xxdk.CMIXParams{}
	err := p.Unmarshal(data)
	return *p, err
}

// parseE2EParams is a helper function which parses a JSON marshalled
// [xxdk.E2EParams].
func parseE2EParams(data []byte) (xxdk.E2EParams, error) {
	p := &xxdk.E2EParams{}
	err := p.Unmarshal(data)
	return *p, err
}
