////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package bindings

import xxdk "gitlab.com/elixxir/client/xxdk2"

// DownloadAndVerifySignedNdfWithUrl retrieves the NDF from a specified URL.
// The NDF is processed into a protobuf containing a signature that is verified
// using the cert string passed in. The NDF is returned as marshaled byte data
// that may be used to start a client.
func DownloadAndVerifySignedNdfWithUrl(url, cert string) ([]byte, error) {
	return xxdk.DownloadAndVerifySignedNdfWithUrl(url, cert)
}
