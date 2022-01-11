///////////////////////////////////////////////////////////////////////////////
// Copyright © 2021 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"io/ioutil"
	"net/http"
)

// DownloadBannedUsers returns a byte array representing banned user IDs in CSV format.
// See https://git.xx.network/elixxir/banned-users.
func DownloadBannedUsers() ([]byte, error) {
	// Build a request for the file
	resp, err := http.Get("https://elixxir-bins.s3-us-west-1.amazonaws.com/client/banned/bannedClientList.csv")
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
