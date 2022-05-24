///////////////////////////////////////////////////////////////////////////////
// Copyright © 2021 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package old

import (
	"io/ioutil"
	"net/http"
)

// DownloadErrorDB returns a []byte containing the JSON data
// describing client errors.
// See https://git.xx.network/elixxir/client-error-database/
func DownloadErrorDB() ([]byte, error) {
	// Build a request for the file
	resp, err := http.Get("https://elixxir-bins.s3-us-west-1.amazonaws.com/client/errors/clientErrors.json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Download the contents of the file
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Return it to the user
	return content, nil
}

// DownloadDAppRegistrationDB returns a []byte containing
// the JSON data describing registered dApps.
// See https://git.xx.network/elixxir/registered-dapps
func DownloadDAppRegistrationDB() ([]byte, error) {
	// Build a request for the file
	resp, err := http.Get("https://elixxir-bins.s3-us-west-1.amazonaws.com/client/dapps/appdb.json")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Download the contents of the file
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Return it to the user
	return content, nil
}
