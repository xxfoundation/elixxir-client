///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package backup

import (
	"crypto/rand"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStoreAndLoad(t *testing.T) {

	fakeRSAKay := make([]byte, 4096/8)
	_, err := rand.Read(fakeRSAKay)
	require.NoError(t, err)

	backup := &Backup{
		TransmissionIdentity: TransmissionIdentity{
			RSASigningPrivateKey: fakeRSAKay,
		},
	}

	key := make([]byte, 32)
	_, err = rand.Read(key)
	require.NoError(t, err)

	nonce := make([]byte, 24)
	_, err = rand.Read(nonce)
	require.NoError(t, err)

	dir, err := ioutil.TempDir("", "backup_state_test")
	require.NoError(t, err)

	filepath := filepath.Join(dir, "tmpfile")

	err = backup.Store(filepath, key, nonce)
	require.NoError(t, err)

	newbackup := &Backup{}
	err = newbackup.Load(filepath, key)
	require.NoError(t, err)

	require.Equal(t, newbackup.TransmissionIdentity.RSASigningPrivateKey, backup.TransmissionIdentity.RSASigningPrivateKey)
}
