////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package dm

import (
	"encoding/base64"
	"github.com/stretchr/testify/require"
	"gitlab.com/elixxir/crypto/codename"
	"gitlab.com/elixxir/crypto/nike/ecdh"
	"gitlab.com/xx_network/crypto/csprng"
	"strings"
	"testing"
)

// Tests that a URL created via dmClient.ShareURL can be decoded using
// DecodeShareURL and that it matches the original.
func TestDmClient_ShareURL_DecodeShareURL(t *testing.T) {
	host := "https://internet.speakeasy.tech/"
	rng := csprng.NewSystemRNG()

	// Construct dm Client w/ minimum required values
	me, _ := codename.GenerateIdentity(rng)
	privateKey := ecdh.Edwards2EcdhNikePrivateKey(me.Privkey)
	publicKey := ecdh.ECDHNIKE.DerivePublicKey(privateKey)

	// Construct URL
	url, err := ShareURL(host, 5, int32(me.GetDMToken()), publicKey, rng)
	require.NoError(t, err)

	// Decode URL
	receivedToken, receivedPubKey, err := DecodeShareURL(url, "")
	require.NoError(t, err)

	// Check that the decoded values match the original values
	require.Equal(t, int32(me.GetDMToken()), receivedToken)
	require.Equal(t, publicKey, receivedPubKey)
}

// Smoke test of DecodeShareURL.
func TestDmClient_ShareURL(t *testing.T) {
	url := "https://internet.speakeasy.tech/?l=32&m=5&p=EfDzQDa4fQ5BoqNIMbECFDY9ckRr_fadd8F1jE49qJc%3D&t=4231817746&v=1"
	dmToken, pubKey, err := DecodeShareURL(url, "")
	require.NoError(t, err)

	t.Logf("dmToken: %d", dmToken)
	t.Logf("RsaPubKey: %s", base64.URLEncoding.EncodeToString(pubKey.Bytes()))
}

// Error path: Tests that dmClient.ShareURL returns an error for an invalid host.
func TestDmClient_ShareURL_ParseError(t *testing.T) {
	// Construct dm Client w/ minimum required values
	rng := csprng.NewSystemRNG()
	me, _ := codename.GenerateIdentity(rng)
	privateKey := ecdh.Edwards2EcdhNikePrivateKey(me.Privkey)
	publicKey := ecdh.ECDHNIKE.DerivePublicKey(privateKey)

	// Attempt to share with an invalid host URL
	host := "invalidHost\x7f"
	expectedErr := strings.Split(parseHostUrlErr, "%")[0]
	_, err := ShareURL(host, 10, int32(me.GetDMToken()), publicKey, rng)
	require.Error(t, err)
	require.Contains(t, err.Error(), expectedErr)
}

// Error path: Tests that DecodeShareURL returns an error for an invalid host.
func TestDecodeShareURL_ParseError(t *testing.T) {
	host := "invalidHost\x7f"
	expectedErr := strings.Split(parseShareUrlErr, "%")[0]

	_, _, err := DecodeShareURL(host, "")
	require.Error(t, err)
	require.Contains(t, err.Error(), expectedErr)
}

// Error path: Tests that DecodeShareURL returns errors for a list of invalid
// URLs.
func TestDecodeShareURL_DecodeError(t *testing.T) {
	type test struct {
		url, password, err string
	}

	tests := []test{
		{"test?", "", urlVersionErr},
		{"test?v=q", "", parseVersionErr},
		{"test?v=2", "", versionErr},
	}

	for _, tt := range tests {
		expected := strings.Split(tt.err, "%")[0]

		_, _, err := DecodeShareURL(tt.url, tt.password)
		require.Error(t, err)
		require.Contains(t, err.Error(), expected)
	}
}
