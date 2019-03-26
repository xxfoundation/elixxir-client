////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package crypto

import (
	"bytes"
	"gitlab.com/elixxir/client/user"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/e2e"
	cmix "gitlab.com/elixxir/crypto/messaging"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/id"
	"testing"
)

var PRIME = "FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1" +
	"29024E088A67CC74020BBEA63B139B22514A08798E3404DD" +
	"EF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245" +
	"E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7ED" +
	"EE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3D" +
	"C2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F" +
	"83655D23DCA3AD961C62F356208552BB9ED529077096966D" +
	"670C354E4ABC9804F1746C08CA18217C32905E462E36CE3B" +
	"E39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9" +
	"DE2BCBF6955817183995497CEA956AE515D2261898FA0510" +
	"15728E5A8AAAC42DAD33170D04507A33A85521ABDF1CBA64" +
	"ECFB850458DBEF0A8AEA71575D060C7DB3970F85A6E1E4C7" +
	"ABF5AE8CDB0933D71E8C94E04A25619DCEE3D2261AD2EE6B" +
	"F12FFA06D98A0864D87602733EC86A64521F2B18177B200C" +
	"BBE117577A615D6C770988C0BAD946E208E24FA074E5AB31" +
	"43DB5BFCE0FD108E4B82D120A92108011A723C12A787E6D7" +
	"88719A10BDBA5B2699C327186AF4E23C1A946834B6150BDA" +
	"2583E9CA2AD44CE8DBBBC2DB04DE8EF92E8EFC141FBECAA6" +
	"287C59474E6BC05D99B2964FA090C3A2233BA186515BE7ED" +
	"1F612970CEE2D7AFB81BDD762170481CD0069127D5B05AA9" +
	"93B4EA988D8FDDC186FFB7DC90A6C08F4DF435C934063199" +
	"FFFFFFFFFFFFFFFF"
var salt = []byte(
	"fdecfa52a8ad1688dbfa7d16df74ebf27e535903c469cefc007ebbe1ee895064")

func setup(t *testing.T) {
	// Init Grp var
	InitCrypto()

	u, _ := user.Users.GetUser(id.NewUserFromUint(1, t))

	nk := make([]user.NodeKeys, 1)

	for i := range nk {
		// transmission and reception keys need to be inverses of each other.
		// this makes it possible for the reception key to decrypt the
		// transmission key without spinning up a whole server to decouple them

		nk[i].TransmissionKeys.Base = Grp.NewInt(1)
		nk[i].TransmissionKeys.Recursive = Grp.NewIntFromString(
			"ad333f4ccea0ccf2afcab6c1b9aa2384e561aee970046e39b7f2a78c3942a251", 16)
		nk[i].ReceptionKeys.Base = Grp.NewInt(1)
		nk[i].ReceptionKeys.Recursive = Grp.Inverse(
			nk[i].TransmissionKeys.Recursive, Grp.NewInt(1))
	}
	user.TheSession = user.NewSession(u, "", nk, nil)
}

func TestEncryptDecrypt(t *testing.T) {
	setup(t)

	sender := id.NewUserFromUint(38, t)
	recipient := id.NewUserFromUint(29, t)
	msg := format.NewMessage()
	msg.SetSender(sender)
	msg.SetRecipient(recipient)
	msgPayload := []byte("help me, i'm stuck in an" +
		" EnterpriseTextLabelDescriptorSetPipelineStateFactoryBeanFactory")
	msg.SetPayloadData(msgPayload)

	// Generate a compound encryption key
	encryptionKey := Grp.NewInt(1)
	for _, key := range user.TheSession.GetKeys() {
		baseKey := key.TransmissionKeys.Base
		partialEncryptionKey := cmix.NewEncryptionKey(salt, baseKey, Grp)
		Grp.Mul(encryptionKey, encryptionKey, partialEncryptionKey)
		//TODO: Add KMAC generation here
	}

	decryptionKey := Grp.NewMaxInt()
	Grp.Inverse(encryptionKey, decryptionKey)

	// do the encryption and the decryption
	e2eKey := e2e.Keygen(Grp, nil, nil)
	assocData, payload := Encrypt(encryptionKey, Grp, msg, e2eKey)
	encryptedNet := &pb.CmixMessage{
		SenderID:       sender.Bytes(),
		MessagePayload: payload,
		AssociatedData: assocData,
	}
	decrypted, err := Decrypt(decryptionKey, Grp, encryptedNet)

	if err != nil {
		t.Fatalf("Couldn't decrypt message: %v", err.Error())
	}
	if *decrypted.GetSender() != *sender {
		t.Errorf("Sender differed from expected: Got %q, expected %q",
			decrypted.GetRecipient(), sender)
	}
	if *decrypted.GetRecipient() != *recipient {
		t.Errorf("Recipient differed from expected: Got %q, expected %q",
			decrypted.GetRecipient(), sender)
	}
	if !bytes.Equal(decrypted.GetPayloadData(), msgPayload) {
		t.Errorf("Decrypted payload differed from expected: Got %q, "+
			"expected %q", decrypted.GetPayloadData(), msgPayload)
	}
}
