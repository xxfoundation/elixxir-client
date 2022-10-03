package ctidh

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNike(t *testing.T) {
	alicePrivateKey, alicePublicKey := CTIDHNIKE.NewKeypair()
	bobPrivateKey, bobPublicKey := CTIDHNIKE.NewKeypair()

	secret1 := alicePrivateKey.DeriveSecret(bobPublicKey)
	secret2 := bobPrivateKey.DeriveSecret(alicePublicKey)

	require.Equal(t, secret1, secret2)
}

func TestPrivateKeyMarshaling(t *testing.T) {
	alicePrivateKey, _ := CTIDHNIKE.NewKeypair()

	alicePrivateKeyBytes := alicePrivateKey.Bytes()
	alice2PrivateKey, _ := CTIDHNIKE.NewKeypair()

	err := alice2PrivateKey.FromBytes(alicePrivateKeyBytes)
	require.NoError(t, err)

	alice2PrivateKeyBytes := alice2PrivateKey.Bytes()

	require.Equal(t, alice2PrivateKeyBytes, alicePrivateKeyBytes)

	alice3PrivateKey, err := CTIDHNIKE.UnmarshalBinaryPrivateKey(alice2PrivateKeyBytes)
	require.NoError(t, err)

	alice3PrivateKeyBytes := alice3PrivateKey.Bytes()

	require.Equal(t, alice3PrivateKeyBytes, alice2PrivateKeyBytes)
}

func TestPublicKeyMarshaling(t *testing.T) {
	_, alicePublicKey := CTIDHNIKE.NewKeypair()

	alicePublicKeyBytes := alicePublicKey.Bytes()
	_, alice2PublicKey := CTIDHNIKE.NewKeypair()

	err := alice2PublicKey.FromBytes(alicePublicKeyBytes)
	require.NoError(t, err)

	alice2PublicKeyBytes := alice2PublicKey.Bytes()

	require.Equal(t, alice2PublicKeyBytes, alicePublicKeyBytes)

	alice3PublicKey, err := CTIDHNIKE.UnmarshalBinaryPublicKey(alice2PublicKeyBytes)
	require.NoError(t, err)

	alice3PublicKeyBytes := alice3PublicKey.Bytes()

	require.Equal(t, alice3PublicKeyBytes, alice2PublicKeyBytes)
}
