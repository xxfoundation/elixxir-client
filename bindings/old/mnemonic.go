///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package old

import "gitlab.com/elixxir/client/api"

// StoreSecretWithMnemonic stores the secret tied with the mnemonic to storage.
// Unlike other storage operations, this does not use EKV, as that is
// intrinsically tied to client operations, which the user will not have while
// trying to recover their account. As such, we store the encrypted data
// directly, with a specified path. Path will be a valid filepath in which the
// recover file will be stored as ".recovery".
//
// As an example, given "home/user/xxmessenger/storagePath",
// the recovery file will be stored at
// "home/user/xxmessenger/storagePath/.recovery"
func StoreSecretWithMnemonic(secret []byte, path string) (string, error) {
	return api.StoreSecretWithMnemonic(secret, path)
}

// LoadSecretWithMnemonic loads the secret stored from the call to
// StoreSecretWithMnemonic. The path given should be the same filepath
// as the path given in StoreSecretWithMnemonic. There should be a file
// in this path called ".recovery". This operation is not tied
// to client operations, as the user will not have a client when trying to
// recover their account.
func LoadSecretWithMnemonic(mnemonic, path string) (secret []byte, err error) {
	return api.LoadSecretWithMnemonic(mnemonic, path)
}
