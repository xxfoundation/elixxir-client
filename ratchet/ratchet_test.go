package ratchet

import (
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/elixxir/client/ctidh"
)

func TestRatchet(t *testing.T) {

	nike := ctidh.NewCTIDHNIKE()
	alicePrivateKey, alicePublicKey := nike.NewKeypair()
	bobPrivateKey, bobPublicKey := nike.NewKeypair()

	scheme := NewScheme()
	aliceRatchetSend := scheme.New(alicePrivateKey, bobPublicKey)
	aliceRatchetReceive := scheme.New(alicePrivateKey, bobPublicKey)

	bobRatchetSend := scheme.New(bobPrivateKey, alicePublicKey)
	bobRatchetReceive := scheme.New(bobPrivateKey, alicePublicKey)

	msg1 := []byte("hello")
	msg2 := []byte("hi hi")

	encryptedMessage1 := aliceRatchetSend.Encrypt(msg1)
	plaintext1, err := bobRatchetReceive.Decrypt(encryptedMessage1)
	require.NoError(t, err)
	require.Equal(t, plaintext1, msg1)

	encryptedMessage2 := bobRatchetSend.Encrypt(msg1)
	plaintext2, err := aliceRatchetReceive.Decrypt(encryptedMessage2)
	require.NoError(t, err)
	require.Equal(t, plaintext2, msg2)
}
