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
	"gitlab.com/elixxir/primitives/circuit"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/id"
	"golang.org/x/crypto/blake2b"
	"os"
	"testing"
	"time"
)

const numNodes = 5

var salt = []byte(
	"fdecfa52a8ad1688dbfa7d16df74ebf27e535903c469cefc007ebbe1ee895064")

var session user.Session
var serverPayloadAKey *cyclic.Int
var serverPayloadBKey *cyclic.Int

var topology *circuit.Circuit

func setup() {

	cmixGrp, e2eGrp := getGroups()

	user.InitUserRegistry(cmixGrp)

	UID := id.NewUserFromUints(&[4]uint64{0, 0, 0, 18})
	u, _ := user.Users.GetUser(UID)

	var nodeSlice []*id.Node

	//build topology
	for i := 0; i < numNodes; i++ {
		nodeBytes := make([]byte, id.NodeIdLen)
		nodeBytes[0] = byte(i)
		nodeId := id.NewNodeFromBytes(nodeBytes)
		nodeSlice = append(nodeSlice, nodeId)
	}

	topology = circuit.New(nodeSlice)

	nkMap := make(map[id.Node]user.NodeKeys)

	tempKey := cmixGrp.NewInt(1)
	serverPayloadAKey = cmixGrp.NewInt(1)
	serverPayloadBKey = cmixGrp.NewInt(1)

	h, _ := blake2b.New256(nil)

	for i := 0; i < numNodes; i++ {

		nk := user.NodeKeys{}

		h.Reset()
		h.Write(salt)

		nk.TransmissionKey = cmixGrp.NewInt(int64(2 + i))
		cmix.NodeKeyGen(cmixGrp, salt, nk.TransmissionKey, tempKey)
		cmixGrp.Mul(serverPayloadAKey, tempKey, serverPayloadAKey)

		cmix.NodeKeyGen(cmixGrp, h.Sum(nil), nk.TransmissionKey, tempKey)
		cmixGrp.Mul(serverPayloadBKey, tempKey, serverPayloadBKey)

		nkMap[*topology.GetNodeAtIndex(i)] = nk
	}

	session = user.NewSession(nil, u, nkMap,
		nil, nil, nil, nil, cmixGrp, e2eGrp)
}

func TestMain(m *testing.M) {
	setup()
	os.Exit(m.Run())
}

func TestFullEncryptDecrypt(t *testing.T) {
	cmixGrp, e2eGrp := getGroups()

	sender := id.NewUserFromUint(38, t)
	recipient := id.NewUserFromUint(29, t)
	msg := format.NewMessage()
	msg.SetRecipient(recipient)
	msgPayload := []byte("help me, i'm stuck in an" +
		" EnterpriseTextLabelDescriptorSetPipelineStateFactoryBeanFactory")
	// Normally, msgPayload would be the right length due to padding
	//msgPayload = append(msgPayload, make([]byte,
	//	format.ContentsLen-len(msgPayload)-format.PadMinLen)...)
	msg.Contents.SetRightAligned(msgPayload)
	now := time.Now()
	nowBytes, _ := now.MarshalBinary()
	// Normally, nowBytes would be the right length due to AES encryption
	nowBytes = append(nowBytes, make([]byte, format.TimestampLen-len(nowBytes))...)
	msg.SetTimestamp(nowBytes)

	key := e2eGrp.NewInt(42)
	h, _ := hash.NewCMixHash()
	h.Write(key.Bytes())
	fp := format.Fingerprint{}
	copy(fp[:], h.Sum(nil))

	// E2E Encryption
	E2EEncrypt(e2eGrp, key, fp, msg)

	// CMIX Encryption
	encMsg, _ := CMIXEncrypt(session, topology, salt, msg)

	// Server will decrypt payload (which is OK because the payload is now e2e)
	// This block imitates what the server does during the realtime
	payloadA := cmixGrp.NewIntFromBytes(encMsg.GetPayloadA())
	payloadB := cmixGrp.NewIntFromBytes(encMsg.GetPayloadB())
	// Multiply payloadA and associated data by serverPayloadBkey
	cmixGrp.Mul(payloadA, serverPayloadAKey, payloadA)
	// Multiply payloadB data only by serverPayloadAkey
	cmixGrp.Mul(payloadB, serverPayloadBKey, payloadB)

	decMsg := format.NewMessage()
	decMsg.SetPayloadA(payloadA.LeftpadBytes(uint64(format.PayloadLen)))
	decMsg.SetDecryptedPayloadB(payloadB.LeftpadBytes(uint64(format.PayloadLen)))

	// E2E Decryption
	err := E2EDecrypt(e2eGrp, key, decMsg)

	if err != nil {
		t.Errorf("E2EDecrypt returned error: %v", err.Error())
	}

	if *decMsg.GetRecipient() != *recipient {
		t.Errorf("Recipient differed from expected: Got %q, expected %q",
			decMsg.GetRecipient(), sender)
	}
	if !bytes.Equal(decMsg.Contents.GetRightAligned(), msgPayload) {
		t.Errorf("Decrypted payload differed from expected: Got %q, "+
			"expected %q", decMsg.Contents.Get(), msgPayload)
	}
}

// E2E unsafe functions should only be used when the payload
// to be sent occupies the whole payload structure, i.e. 256 bytes
func TestFullEncryptDecrypt_Unsafe(t *testing.T) {
	cmixGrp, e2eGrp := getGroups()
	sender := id.NewUserFromUint(38, t)
	recipient := id.NewUserFromUint(29, t)
	msg := format.NewMessage()
	msg.SetRecipient(recipient)
	msgPayload := []byte(
		" EnterpriseTextLabelDescriptorSetPipelineStateFactoryBeanFactory" +
			" EnterpriseTextLabelDescriptorSetPipelineStateFactoryBeanFactory" +
			" EnterpriseTextLabelDescriptorSetPipelineStateFactoryBeanFactory" +
			" EnterpriseTextLabelDescriptorSetPipelineStateFactoryBeanFactory" +
			" EnterpriseTextLabelDescriptorSetPipelineStateFactoryBeanFactory" +
			" EnterpriseTextLabelDescriptorSetPipelineStateFactoryBeanFactory" +
			" EnterpriseTextLabelDescriptorSetPipelineStateFactoryBeanFactory")
	msg.Contents.Set(msgPayload[:format.ContentsLen])

	msg.SetTimestamp(make([]byte, 16))

	key := e2eGrp.NewInt(42)
	h, _ := hash.NewCMixHash()
	h.Write(key.Bytes())
	fp := format.Fingerprint{}
	copy(fp[:], h.Sum(nil))

	// E2E Encryption without padding
	E2EEncryptUnsafe(e2eGrp, key, fp, msg)

	// CMIX Encryption
	encMsg, _ := CMIXEncrypt(session, topology, salt, msg)

	// Server will decrypt payload (which is OK because the payload is now e2e)
	// This block imitates what the server does during the realtime
	var encryptedNet *pb.Slot
	{
		payload := cmixGrp.NewIntFromBytes(encMsg.GetPayloadA())
		assocData := cmixGrp.NewIntFromBytes(encMsg.GetPayloadB())
		// Multiply payload and associated data by transmission key only
		cmixGrp.Mul(payload, serverPayloadAKey, payload)
		// Multiply associated data only by transmission key
		cmixGrp.Mul(assocData, serverPayloadBKey, assocData)
		encryptedNet = &pb.Slot{
			SenderID: sender.Bytes(),
			Salt:     salt,
			PayloadA: payload.LeftpadBytes(uint64(format.PayloadLen)),
			PayloadB: assocData.LeftpadBytes(uint64(format.PayloadLen)),
		}
	}

	decMsg := format.NewMessage()
	decMsg.SetPayloadA(encryptedNet.PayloadA)
	decMsg.SetDecryptedPayloadB(encryptedNet.PayloadB)

	// E2E Decryption
	err := E2EDecryptUnsafe(e2eGrp, key, decMsg)

	if err != nil {
		t.Errorf("E2EDecryptUnsafe returned error: %v", err.Error())
	}

	if *decMsg.GetRecipient() != *recipient {
		t.Errorf("Recipient differed from expected: Got %q, expected %q",
			decMsg.GetRecipient(), sender)
	}
	if !bytes.Equal(decMsg.Contents.Get(), msgPayload[:format.ContentsLen]) {
		t.Errorf("Decrypted payload differed from expected: Got %q, "+
			"expected %q", decMsg.Contents.Get(), msgPayload[:format.ContentsLen])
	}
}

// Test that E2EEncrypt panics if the payload is too big (can't be padded)
func TestE2EEncrypt_Panic(t *testing.T) {
	_, e2eGrp := getGroups()
	recipient := id.NewUserFromUint(29, t)
	msg := format.NewMessage()
	msg.SetRecipient(recipient)
	msgPayload := []byte("help me, i'm stuck in an" +
		" EnterpriseTextLabelDescriptorSetPipelineStateFactoryBeanFactory" +
		" EnterpriseTextLabelDescriptorSetPipelineStateFactoryBeanFactory" +
		" EnterpriseTextLabelDescriptorSetPipelineStateFactoryBeanFactory" +
		" EnterpriseTextLabelDescriptorSetPipelineStateFactoryBeanFactory" +
		" EnterpriseTextLabelDescriptorSetPipelineStateFactoryBeanFactory" +
		" EnterpriseTextLabelDescriptorSetPipelineStateFactoryBeanFactory")
	msgPayload = msgPayload[:format.ContentsLen]
	msg.Contents.Set(msgPayload)
	msg.SetTimestamp(make([]byte, 16))

	key := e2eGrp.NewInt(42)
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
	E2EEncrypt(e2eGrp, key, fp, msg)
}

// Test that E2EDecrypt and E2EDecryptUnsafe handle errors correctly
func TestE2EDecrypt_Errors(t *testing.T) {
	_, e2eGrp := getGroups()
	recipient := id.NewUserFromUint(29, t)
	msg := format.NewMessage()
	msg.SetRecipient(recipient)
	msgPayload := []byte("help me, i'm stuck in an EnterpriseTextLabelDescriptorSetPipelineStateFactoryBeanFactory ")
	msg.Contents.SetRightAligned(msgPayload)
	msg.SetTimestamp(make([]byte, 16))

	key := e2eGrp.NewInt(42)
	h, _ := hash.NewCMixHash()
	h.Write(key.Bytes())
	fp := format.Fingerprint{}
	copy(fp[:], h.Sum(nil))

	// E2E Encryption
	E2EEncrypt(e2eGrp, key, fp, msg)

	// Copy message
	badMsg := format.NewMessage()
	badMsg.SetPayloadA(msg.GetPayloadA())
	badMsg.SetPayloadB(msg.GetPayloadB())

	// Corrupt MAC to make decryption fail
	badMsg.SetMAC([]byte("sakfaskfajskasfkkaskfanjffffjnaf"))

	// E2E Decryption returns error
	err := E2EDecrypt(e2eGrp, key, badMsg)

	if err == nil {
		t.Errorf("E2EDecrypt should have returned error")
	} else {
		t.Logf("E2EDecrypt error: %v", err.Error())
	}

	// Unsafe E2E Decryption returns error
	err = E2EDecryptUnsafe(e2eGrp, key, badMsg)

	if err == nil {
		t.Errorf("E2EDecryptUnsafe should have returned error")
	} else {
		t.Logf("E2EDecryptUnsafe error: %v", err.Error())
	}

	// Set correct MAC again
	badMsg.SetMAC(msg.GetMAC())

	// Corrupt timestamp to make decryption fail
	badMsg.SetTimestamp([]byte("ABCDEF1234567890"))

	// E2E Decryption returns error
	err = E2EDecrypt(e2eGrp, key, badMsg)

	if err == nil {
		t.Errorf("E2EDecrypt should have returned error")
	} else {
		t.Logf("E2EDecrypt error: %v", err.Error())
	}

	// Unsafe E2E Decryption returns error
	err = E2EDecryptUnsafe(e2eGrp, key, badMsg)

	if err == nil {
		t.Errorf("E2EDecryptUnsafe should have returned error")
	} else {
		t.Logf("E2EDecryptUnsafe error: %v", err.Error())
	}

	// Set correct Timestamp again
	badMsg.SetTimestamp(msg.GetTimestamp())

	// Corrupt payload to make decryption fail
	badMsg.Contents.SetRightAligned([]byte(
		"sakomnsfjeiknheuijhgfyaistuajhfaiuojfkhufijsahufiaij"))

	// Calculate new MAC to avoid failing on that verification again
	newMAC := hash.CreateHMAC(badMsg.Contents.Get(), key.Bytes())
	badMsg.SetMAC(newMAC)

	// E2E Decryption returns error
	err = E2EDecrypt(e2eGrp, key, badMsg)

	if err == nil {
		t.Errorf("E2EDecrypt should have returned error")
	} else {
		t.Logf("E2EDecrypt error: %v", err.Error())
	}
}

func getGroups() (*cyclic.Group, *cyclic.Group) {

	cmixGrp := cyclic.NewGroup(
		large.NewIntFromString("FFFFFFFFFFFFFFFFC90FDAA22168C234C4C6628B80DC1CD1"+
			"29024E088A67CC74020BBEA63B139B22514A08798E3404DD"+
			"EF9519B3CD3A431B302B0A6DF25F14374FE1356D6D51C245"+
			"E485B576625E7EC6F44C42E9A637ED6B0BFF5CB6F406B7ED"+
			"EE386BFB5A899FA5AE9F24117C4B1FE649286651ECE45B3D"+
			"C2007CB8A163BF0598DA48361C55D39A69163FA8FD24CF5F"+
			"83655D23DCA3AD961C62F356208552BB9ED529077096966D"+
			"670C354E4ABC9804F1746C08CA18217C32905E462E36CE3B"+
			"E39E772C180E86039B2783A2EC07A28FB5C55DF06F4C52C9"+
			"DE2BCBF6955817183995497CEA956AE515D2261898FA0510"+
			"15728E5A8AACAA68FFFFFFFFFFFFFFFF", 16),
		large.NewIntFromString("2", 16),
		large.NewIntFromString("2", 16))

	e2eGrp := cyclic.NewGroup(
		large.NewIntFromString("E2EE983D031DC1DB6F1A7A67DF0E9A8E5561DB8E8D49413394C049B"+
			"7A8ACCEDC298708F121951D9CF920EC5D146727AA4AE535B0922C688B55B3DD2AE"+
			"DF6C01C94764DAB937935AA83BE36E67760713AB44A6337C20E7861575E745D31F"+
			"8B9E9AD8412118C62A3E2E29DF46B0864D0C951C394A5CBBDC6ADC718DD2A3E041"+
			"023DBB5AB23EBB4742DE9C1687B5B34FA48C3521632C4A530E8FFB1BC51DADDF45"+
			"3B0B2717C2BC6669ED76B4BDD5C9FF558E88F26E5785302BEDBCA23EAC5ACE9209"+
			"6EE8A60642FB61E8F3D24990B8CB12EE448EEF78E184C7242DD161C7738F32BF29"+
			"A841698978825B4111B4BC3E1E198455095958333D776D8B2BEEED3A1A1A221A6E"+
			"37E664A64B83981C46FFDDC1A45E3D5211AAF8BFBC072768C4F50D7D7803D2D4F2"+
			"78DE8014A47323631D7E064DE81C0C6BFA43EF0E6998860F1390B5D3FEACAF1696"+
			"015CB79C3F9C2D93D961120CD0E5F12CBB687EAB045241F96789C38E89D796138E"+
			"6319BE62E35D87B1048CA28BE389B575E994DCA755471584A09EC723742DC35873"+
			"847AEF49F66E43873", 16),
		large.NewIntFromString("2", 16),
		large.NewIntFromString("2", 16))

	return cmixGrp, e2eGrp

}
