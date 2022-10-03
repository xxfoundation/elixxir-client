package ratchet

/*
func TestXXRatchetEncryptDecrypt(t *testing.T) {
	salt := make([]byte, 32)
	_, err := rand.Read(salt)
	require.NoError(t, err)

	size := uint32(32)

	alicePrivKey, alicePubKey := DefaultNIKE.NewKeypair()
	bobPrivKey, bobPubKey := DefaultNIKE.NewKeypair()

	aliceRatchet := DefaultXXRatchetFactory.NewXXRatchet(alicePrivKey, alicePubKey, bobPubKey)
	bobRatchet := DefaultXXRatchetFactory.NewXXRatchet(bobPrivKey, bobPubKey, alicePubKey)
	aliceRatchet.Encrypt()

}
*/
