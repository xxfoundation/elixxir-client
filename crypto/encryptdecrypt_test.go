////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package crypto_test

import (
	"bytes"
	"gitlab.com/elixxir/client/crypto"
	"gitlab.com/elixxir/client/user"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/cmix"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/id"
	"testing"
)

var salt = []byte(
	"fdecfa52a8ad1688dbfa7d16df74ebf27e535903c469cefc007ebbe1ee895064")

func setup(t *testing.T) {
	base := 16

	pString := "9DB6FB5951B66BB6FE1E140F1D2CE5502374161FD6538DF1648218642F0B5C48" +
		"C8F7A41AADFA187324B87674FA1822B00F1ECF8136943D7C55757264E5A1A44F" +
		"FE012E9936E00C1D3E9310B01C7D179805D3058B2A9F4BB6F9716BFE6117C6B5" +
		"B3CC4D9BE341104AD4A80AD6C94E005F4B993E14F091EB51743BF33050C38DE2" +
		"35567E1B34C3D6A5C0CEAA1A0F368213C3D19843D0B4B09DCB9FC72D39C8DE41" +
		"F1BF14D4BB4563CA28371621CAD3324B6A2D392145BEBFAC748805236F5CA2FE" +
		"92B871CD8F9C36D3292B5509CA8CAA77A2ADFC7BFD77DDA6F71125A7456FEA15" +
		"3E433256A2261C6A06ED3693797E7995FAD5AABBCFBE3EDA2741E375404AE25B"

	gString := "5C7FF6B06F8F143FE8288433493E4769C4D988ACE5BE25A0E24809670716C613" +
		"D7B0CEE6932F8FAA7C44D2CB24523DA53FBE4F6EC3595892D1AA58C4328A06C4" +
		"6A15662E7EAA703A1DECF8BBB2D05DBE2EB956C142A338661D10461C0D135472" +
		"085057F3494309FFA73C611F78B32ADBB5740C361C9F35BE90997DB2014E2EF5" +
		"AA61782F52ABEB8BD6432C4DD097BC5423B285DAFB60DC364E8161F4A2A35ACA" +
		"3A10B1C4D203CC76A470A33AFDCBDD92959859ABD8B56E1725252D78EAC66E71" +
		"BA9AE3F1DD2487199874393CD4D832186800654760E1E34C09E4D155179F9EC0" +
		"DC4473F996BDCE6EED1CABED8B6F116F7AD9CF505DF0F998E34AB27514B0FFE7"

	qString := "F2C3119374CE76C9356990B465374A17F23F9ED35089BD969F61C6DDE9998C1F"

	p := large.NewIntFromString(pString, base)
	g := large.NewIntFromString(gString, base)
	q := large.NewIntFromString(qString, base)

	grp := cyclic.NewGroup(p, g, q)

	u, _ := user.Users.GetUser(id.NewUserFromUint(1, t))

	nk := make([]user.NodeKeys, 1)

	for i := range nk {
		// transmission and reception keys need to be inverses of each other.
		// this makes it possible for the reception key to decrypt the
		// transmission key without spinning up a whole server to decouple them

		nk[i].TransmissionKeys.Base = grp.NewInt(1)
		nk[i].TransmissionKeys.Recursive = grp.NewIntFromString(
			"ad333f4ccea0ccf2afcab6c1b9aa2384e561aee970046e39b7f2a78c3942a251", 16)
		nk[i].ReceptionKeys.Base = grp.NewInt(1)
		nk[i].ReceptionKeys.Recursive = grp.Inverse(
			nk[i].TransmissionKeys.Recursive, grp.NewInt(1))
	}
	user.TheSession = user.NewSession(u, "", nk, nil, nil, grp)
}

func TestEncryptDecrypt(t *testing.T) {
	setup(t)

	grp := user.TheSession.GetGroup()
	sender := id.NewUserFromUint(38, t)
	recipient := id.NewUserFromUint(29, t)
	msg := format.NewMessage()
	msg.SetSender(sender)
	msg.SetRecipient(recipient)
	msgPayload := []byte("help me, i'm stuck in an" +
		" EnterpriseTextLabelDescriptorSetPipelineStateFactoryBeanFactory")
	msg.SetPayloadData(msgPayload)

	// Generate a compound encryption key
	encryptionKey := grp.NewInt(1)
	for _, key := range user.TheSession.GetKeys() {
		baseKey := key.TransmissionKeys.Base
		partialEncryptionKey := cmix.NewEncryptionKey(salt, baseKey, grp)
		grp.Mul(encryptionKey, encryptionKey, partialEncryptionKey)
		//TODO: Add KMAC generation here
	}

	decryptionKey := grp.NewMaxInt()
	grp.Inverse(encryptionKey, decryptionKey)

	// do the encryption and the decryption
	e2eKey := e2e.Keygen(grp, nil, nil)
	assocData, payload := crypto.Encrypt(encryptionKey, grp, msg, e2eKey)
	encryptedNet := &pb.CmixMessage{
		SenderID:       sender.Bytes(),
		MessagePayload: payload,
		AssociatedData: assocData,
	}
	decrypted, err := crypto.Decrypt(decryptionKey, grp, encryptedNet)

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
