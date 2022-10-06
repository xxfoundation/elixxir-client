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
	threshhold := uint32(20)

	scheme := DefaultSymmetricKeyRatchetFactory
	aliceRatchetSend := scheme.New(sharedSecret, salt, size, threshhold)
	aliceRatchetReceive := scheme.New(sharedSecret, salt, size, threshhold)

	bobRatchetSend := scheme.New(sharedSecret, salt, size, threshhold)
	bobRatchetReceive := scheme.New(sharedSecret, salt, size, threshhold)

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
	threshhold := uint32(20)

	scheme := DefaultSymmetricKeyRatchetFactory
	r1 := scheme.New(sharedSecret, salt, size, threshhold)
	blob1, err := r1.Save()
	require.NoError(t, err)

	r2, err := scheme.FromBytes(blob1)
	require.NoError(t, err)
	require.Equal(t, r1, r2)
}

func TestReceiveSendRatchetEncryptDecrypt(t *testing.T) {
	salt := make([]byte, 32)
	_, err := rand.Read(salt)
	require.NoError(t, err)

	size := uint32(32)
	threshhold := uint32(20)

	alicePrivateKey, alicePublicKey := DefaultNIKE.NewKeypair()
	bobPrivateKey, bobPublicKey := DefaultNIKE.NewKeypair()

	aliceReceiveRatchet := NewReceiveRatchet(alicePrivateKey, bobPublicKey,
		salt, size, threshhold)
	aliceSendRatchet := NewSendRatchet(alicePrivateKey, alicePublicKey,
		bobPublicKey, salt, size, threshhold)

	bobReceiveRatchet := NewReceiveRatchet(bobPrivateKey, alicePublicKey,
		salt, size, threshhold)
	bobSendRatchet := NewSendRatchet(bobPrivateKey, bobPublicKey,
		alicePublicKey, salt, size, threshhold)

	msg1 := []byte("hi bob")

	ciphertext1, err := aliceSendRatchet.Encrypt(msg1)
	require.NoError(t, err)

	plaintext1, err := bobReceiveRatchet.Decrypt(ciphertext1)
	require.NoError(t, err)
	require.Equal(t, plaintext1, msg1)

	msg2 := []byte("acknowledged")

	ciphertext2, err := bobSendRatchet.Encrypt(msg2)
	require.NoError(t, err)

	plaintext2, err := aliceReceiveRatchet.Decrypt(ciphertext2)
	require.NoError(t, err)
	require.Equal(t, plaintext2, msg2)
}
