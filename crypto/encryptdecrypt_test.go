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
	"gitlab.com/elixxir/crypto/cmix"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/id"
	"os"
	"testing"
	"time"
)

var salt = []byte(
	"fdecfa52a8ad1688dbfa7d16df74ebf27e535903c469cefc007ebbe1ee895064")

var session user.Session
var serverTransmissionKey *cyclic.Int
var serverReceptionKey *cyclic.Int

func setup() {
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

	UID := new(id.User).SetUints(&[4]uint64{0, 0, 0, 18})
	u, _ := user.Users.GetUser(UID)

	nk := make([]user.NodeKeys, 5)

	baseKey := grp.NewInt(1)
	serverTransmissionKey = grp.NewInt(1)
	serverReceptionKey = grp.NewInt(1)

	for i := range nk {

		nk[i].TransmissionKey = grp.NewInt(int64(2 + i))
		cmix.NodeKeyGen(grp, salt, nk[i].TransmissionKey, baseKey)
		grp.Mul(serverTransmissionKey, baseKey, serverTransmissionKey)
		nk[i].ReceptionKey = grp.NewInt(int64(1000 + i))
		cmix.NodeKeyGen(grp, salt, nk[i].ReceptionKey, baseKey)
		grp.Mul(serverReceptionKey, baseKey, serverReceptionKey)
	}
	session = user.NewSession(nil, u, "", nk,
		nil, nil, grp)
}

func TestMain(m *testing.M) {
	setup()
	os.Exit(m.Run())
}

func TestFullEncryptDecrypt(t *testing.T) {
	grp := session.GetGroup()
	sender := id.NewUserFromUint(38, t)
	recipient := id.NewUserFromUint(29, t)
	msg := format.NewMessage()
	msg.SetSender(sender)
	msg.SetRecipient(recipient)
	msgPayload := []byte("help me, i'm stuck in an" +
		" EnterpriseTextLabelDescriptorSetPipelineStateFactoryBeanFactory")
	msg.SetPayloadData(msgPayload)
	now := time.Now()
	nowBytes, _ := now.MarshalBinary()
	msg.SetTimestamp(nowBytes)

	key := grp.NewInt(42)
	h, _ := hash.NewCMixHash()
	h.Write(key.Bytes())
	fp := format.Fingerprint{}
	copy(fp[:], h.Sum(nil))

	// E2E Encryption
	E2EEncrypt(key, fp, grp, msg)

	// CMIX Encryption
	encMsg := CMIXEncrypt(session, salt, msg)

	// Server will decrypt and re-encrypt payload
	payload := grp.NewIntFromBytes(encMsg.SerializePayload())
	assocData := grp.NewIntFromBytes(encMsg.SerializeAssociatedData())
	// Multiply payload by transmission and reception keys
	grp.Mul(payload, serverTransmissionKey, payload)
	grp.Mul(payload, serverReceptionKey, payload)
	// Multiply associated data only by transmission key
	grp.Mul(assocData, serverTransmissionKey, assocData)
	encryptedNet := &pb.CmixMessage{
		SenderID:       sender.Bytes(),
		Salt:           salt,
		MessagePayload: payload.LeftpadBytes(uint64(format.TOTAL_LEN)),
		AssociatedData: assocData.LeftpadBytes(uint64(format.TOTAL_LEN)),
	}

	// CMIX Decryption
	decMsg := CMIXDecrypt(session, encryptedNet)

	// E2E Decryption
	err := E2EDecrypt(key, grp, decMsg)

	if err != nil {
		t.Errorf("E2EDecrypt returned error: %v", err.Error())
	}

	if *decMsg.GetSender() != *sender {
		t.Errorf("Sender differed from expected: Got %q, expected %q",
			decMsg.GetRecipient(), sender)
	}
	if *decMsg.GetRecipient() != *recipient {
		t.Errorf("Recipient differed from expected: Got %q, expected %q",
			decMsg.GetRecipient(), sender)
	}
	if !bytes.Equal(decMsg.GetPayloadData(), msgPayload) {
		t.Errorf("Decrypted payload differed from expected: Got %q, "+
			"expected %q", decMsg.GetPayloadData(), msgPayload)
	}
}

// Test that E2EEncrypt panics if the payload is too big (can't be padded)
func TestE2EEncrypt_Panic(t *testing.T) {
	grp := session.GetGroup()
	sender := id.NewUserFromUint(38, t)
	recipient := id.NewUserFromUint(29, t)
	msg := format.NewMessage()
	msg.SetSender(sender)
	msg.SetRecipient(recipient)
	msgPayload := []byte("help me, i'm stuck in an" +
		" EnterpriseTextLabelDescriptorSetPipelineStateFactoryBeanFactory" +
		" EnterpriseTextLabelDescriptorSetPipelineStateFactoryBeanFactory" +
		" EnterpriseTextLabelDescriptorSetPipelineStateFactoryBeanFactory" +
		" EnterpriseTextLabelDescriptorSetPipelineStateFactoryBeanFactory")
	msg.SetPayloadData(msgPayload)
	now := time.Now()
	nowBytes, _ := now.MarshalBinary()
	msg.SetTimestamp(nowBytes)

	key := grp.NewInt(42)
	h, _ := hash.NewCMixHash()
	h.Write(key.Bytes())
	fp := format.Fingerprint{}
	copy(fp[:], h.Sum(nil))

	defer func() {
		if r := recover(); r == nil {
			t.Errorf("E2EEncrypt should panic on payload too large")
		}
	}()

	// E2E Encryption Panics
	E2EEncrypt(key, fp, grp, msg)
}

// Test that E2EDecrypt handles errors correctly
func TestE2EDecrypt_Errors(t *testing.T) {
	grp := session.GetGroup()
	sender := id.NewUserFromUint(38, t)
	recipient := id.NewUserFromUint(29, t)
	msg := format.NewMessage()
	msg.SetSender(sender)
	msg.SetRecipient(recipient)
	msgPayload := []byte("help me, i'm stuck in an" +
		" EnterpriseTextLabelDescriptorSetPipelineStateFactoryBeanFactory")
	msg.SetPayloadData(msgPayload)
	now := time.Now()
	nowBytes, _ := now.MarshalBinary()
	msg.SetTimestamp(nowBytes)

	key := grp.NewInt(42)
	h, _ := hash.NewCMixHash()
	h.Write(key.Bytes())
	fp := format.Fingerprint{}
	copy(fp[:], h.Sum(nil))

	// E2E Encryption
	E2EEncrypt(key, fp, grp, msg)

	// Copy message
	badMsg := format.NewMessage()
	badMsg.Payload = format.DeserializePayload(msg.SerializePayload())
	badMsg.AssociatedData = format.DeserializeAssociatedData(msg.SerializeAssociatedData())

	// Corrupt MAC to make decryption fail
	badMsg.SetMAC([]byte("sakfaskfajskasfkkaskfanjjnaf"))

	// E2E Decryption returns error
	err := E2EDecrypt(key, grp, badMsg)

	if err == nil {
		t.Errorf("E2EDecrypt should have returned error")
	} else {
		t.Logf("E2EDecrypt error: %v", err.Error())
	}

	// Set correct MAC again
	badMsg.SetMAC(msg.GetMAC())

	// Corrupt timestamp to make decryption fail
	badMsg.SetTimestamp([]byte("ABCDEF1234567890"))

	// E2E Decryption returns error
	err = E2EDecrypt(key, grp, badMsg)

	if err == nil {
		t.Errorf("E2EDecrypt should have returned error")
	} else {
		t.Logf("E2EDecrypt error: %v", err.Error())
	}

	// Set correct Timestamp again
	badMsg.SetTimestamp(msg.GetTimestamp())

	// Corrupt payload to make decryption fail
	badMsg.SetPayload([]byte("sakomnsfjeiknheuijhgfyaistuajhfaiuojfkhufijsahufiaij"))

	// Calculate new MAC to avoid failing on that verification again
	newMAC := hash.CreateHMAC(badMsg.SerializePayload(), key.Bytes())
	badMsg.SetMAC(newMAC)

	// E2E Decryption returns error
	err = E2EDecrypt(key, grp, badMsg)

	if err == nil {
		t.Errorf("E2EDecrypt should have returned error")
	} else {
		t.Logf("E2EDecrypt error: %v", err.Error())
	}
}
