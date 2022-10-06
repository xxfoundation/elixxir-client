package ratchet

// Major test:
// Create a new ratchet with specific settings, "exhaust" it, and be
// sure the callback is triggered at the appropriate time.

// Property Based Tests
//  1. When Alice initiates an auth channel with Bob, Alice sends
//     auth request to Bob, when Bob decides to confirm:
//     a. Bob creates the relationship using Alice's public key.
//     b. Bob has a sending ratchet in the "triggered"
//     (NewSessionTriggered) list.
//     c. Bob has a receiving ratchet.
//     d. Bob sends a confirmation, if this fails, the sending ratchet
//     is moved to the "created" state.
//  2. Bob sends the confirmation. Bob can resend until a final
//     acknowledgement is received. When Alice receives it:
//     a. Alice creates the relationship using bob's public key.
//     b. Alice has a sending ratchet in the "confirmed" list.
//     c. Alice has a receiving ratchet.
//     d. Alice sends a final auth acknowledgement.
//  3. When Bob receives the auth acknowledgement:
//     a. Bob's sending ratchet is moved to the "confirmed" list.
//  4. When Bob receives a message for a receiving ratchet, and there is only
//     one sending ratchet (i.e., it's new from the auth system, but not
//     acknowledged)
//     a. Bob's only sending ratchet is moved to the "confirmed" list if it is in
//     the "created" or "triggered" state.
//     b. A warning is printed with the state and relationship information.
//  5. When Alice starts to run out of Sending keys in the Sending
//     Ratchet, and decides to send a rekey trigger:
//     a. Alice creates a sending ratchet added to the "sending"
//     list. This uses the pre-existing public key of Bob to derive
//     it's secret.
//     b. If the ratchet is able to be sent, it is moved to the "sent" list.
//     c. If the the ratchet cannot be sent, it is moved to the "confirmed" list.
//  6. When Bob sees the rekey trigger:
//     a. Bob creates a new receiving ratchet with his existing key and
//     Alice's public key.
//     b. Bob sends a confirmation.
//  7. When Alice receive's Bob's confirmation:
//     a. The ratchet is moved from to the "confirmed" state.
//  8. External entities can:
//     a. Move "sent" and "sending" ratchets to unconfirmed.
//     b. Create new relationships like the auth system described in 1-4.
//  9. Sending Messages:
//     a. Ratchets are moved forward, keys are marked as used, then ratchet state
//     is saved.
//  10. Receiving Messages:
//     a. A callback is registered to report new key fingerprints.
//     b. Decryption can fail if a key is reused.

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
