package ratchet

import (
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRatchetEncryptDecrypt(t *testing.T) {
	sharedSecret := make([]byte, 32)
	_, err := rand.Read(sharedSecret)
	require.NoError(t, err)

	salt := make([]byte, 32)
	_, err = rand.Read(salt)
	require.NoError(t, err)

	size := uint32(32)

	scheme := DefaultSymmetricKeyRatchetFactory
	aliceRatchetSend := scheme.New(sharedSecret, salt, size)
	aliceRatchetReceive := scheme.New(sharedSecret, salt, size)

	bobRatchetSend := scheme.New(sharedSecret, salt, size)
	bobRatchetReceive := scheme.New(sharedSecret, salt, size)

	msg1 := []byte("hello")
	msg2 := []byte("hi hi")

	encryptedMessage1, err := aliceRatchetSend.Encrypt(msg1)
	require.NoError(t, err)

	plaintext1, err := bobRatchetReceive.Decrypt(encryptedMessage1)
	require.NoError(t, err)
	require.Equal(t, plaintext1, msg1)

	encryptedMessage2, err := bobRatchetSend.Encrypt(msg2)
	require.NoError(t, err)

	plaintext2, err := aliceRatchetReceive.Decrypt(encryptedMessage2)
	require.NoError(t, err)
	require.Equal(t, plaintext2, msg2)
}

func TestRatchetMarshal(t *testing.T) {
	sharedSecret := make([]byte, 32)
	_, err := rand.Read(sharedSecret)
	require.NoError(t, err)

	salt := make([]byte, 32)
	_, err = rand.Read(salt)
	require.NoError(t, err)

	size := uint32(32)

	scheme := DefaultSymmetricKeyRatchetFactory
	r1 := scheme.New(sharedSecret, salt, size)
	blob1, err := r1.Save()
	require.NoError(t, err)

	r2, err := scheme.FromBytes(blob1)
	require.NoError(t, err)
	require.Equal(t, r1, r2)
}
