///////////////////////////////////////////////////////////////////////////////
// Copyright © 2021 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package old

import (
	"gitlab.com/elixxir/client/api"
)

// DownloadAndVerifySignedNdfWithUrl retrieves the NDF from a specified URL.
// The NDF is processed into a protobuf containing a signature which
// is verified using the cert string passed in. The NDF is returned as marshaled
// byte data which may be used to start a client.
func DownloadAndVerifySignedNdfWithUrl(url, cert string) ([]byte, error) {
	return api.DownloadAndVerifySignedNdfWithUrl(url, cert)
}
