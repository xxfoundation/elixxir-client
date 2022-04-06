////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package fileTransfer

import (
	"fmt"
	"github.com/cloudflare/circl/dh/sidh"
	"github.com/golang/protobuf/proto"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/params"
	util "gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/crypto/diffieHellman"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"reflect"
	"strings"
	"testing"
)

// Tests that the E2E message sent via Manager.sendNewFileTransfer matches
// expected.
func TestManager_sendNewFileTransfer(t *testing.T) {
	m := newTestManager(false, nil, nil, nil, nil, t)

	recipient := id.NewIdFromString("recipient", id.User, t)
	fileName := "testFile"
	fileType := "txt"
	key, _ := ftCrypto.NewTransferKey(NewPrng(42))
	mac := []byte("transferMac")
	numParts, fileSize, retry := uint16(16), uint32(256), float32(1.5)
	preview := []byte("filePreview")

	rng := csprng.NewSystemRNG()
	dhKey := m.store.E2e().GetGroup().NewInt(42)
	pubKey := diffieHellman.GeneratePublicKey(dhKey, m.store.E2e().GetGroup())
	_, mySidhPriv := util.GenerateSIDHKeyPair(sidh.KeyVariantSidhA, rng)
	theirSidhPub, _ := util.GenerateSIDHKeyPair(sidh.KeyVariantSidhB, rng)
	p := params.GetDefaultE2ESessionParams()

	err := m.store.E2e().AddPartner(recipient, pubKey, dhKey,
		mySidhPriv, theirSidhPub, p, p)
	if err != nil {
		t.Errorf("Failed to add partner %s: %+v", recipient, err)
	}

	expected, err := newNewFileTransferE2eMessage(recipient, fileName, fileType,
		key, mac, numParts, fileSize, retry, preview)
	if err != nil {
		t.Errorf("Failed to create new Send message: %+v", err)
	}

	err = m.sendNewFileTransfer(recipient, fileName, fileType, key, mac,
		numParts, fileSize, retry, preview)
	if err != nil {
		t.Errorf("sendNewFileTransfer returned an error: %+v", err)
	}

	received := m.net.(*testNetworkManager).GetE2eMsg(0)

	if !reflect.DeepEqual(expected, received) {
		t.Errorf("Received E2E message does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, received)
	}
}

// Error path: tests that Manager.sendNewFileTransfer returns the expected error
// when SendE2E fails.
func TestManager_sendNewFileTransfer_E2eError(t *testing.T) {
	// Create new test manager with a SendE2E error triggered
	m := newTestManager(true, nil, nil, nil, nil, t)

	recipient := id.NewIdFromString("recipient", id.User, t)
	key, _ := ftCrypto.NewTransferKey(NewPrng(42))

	rng := csprng.NewSystemRNG()
	dhKey := m.store.E2e().GetGroup().NewInt(42)
	pubKey := diffieHellman.GeneratePublicKey(dhKey, m.store.E2e().GetGroup())
	_, mySidhPriv := util.GenerateSIDHKeyPair(sidh.KeyVariantSidhA, rng)
	theirSidhPub, _ := util.GenerateSIDHKeyPair(sidh.KeyVariantSidhB, rng)
	p := params.GetDefaultE2ESessionParams()

	err := m.store.E2e().AddPartner(recipient, pubKey, dhKey,
		mySidhPriv, theirSidhPub, p, p)
	if err != nil {
		t.Errorf("Failed to add partner %s: %+v", recipient, err)
	}

	expectedErr := fmt.Sprintf(newFtSendE2eErr, recipient, "")
	err = m.sendNewFileTransfer(recipient, "", "", key, nil, 16, 256, 1.5, nil)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("sendNewFileTransfer di dnot return the expected error when "+
			"SendE2E failed.\nexpected: %s\nreceived: %+v", expectedErr, err)
	}

	if len(m.net.(*testNetworkManager).e2eMessages) > 0 {
		t.Errorf("%d E2E messeage(s) found when SendE2E should have failed.",
			len(m.net.(*testNetworkManager).e2eMessages))
	}
}

// Tests that newNewFileTransferE2eMessage returns a message.Send with the
// correct recipient and message type and that the payload can be unmarshalled
// into the correct NewFileTransfer.
func Test_newNewFileTransferE2eMessage(t *testing.T) {
	recipient := id.NewIdFromString("recipient", id.User, t)
	key, _ := ftCrypto.NewTransferKey(NewPrng(42))
	expected := &NewFileTransfer{
		FileName:    "testFile",
		FileType:    "txt",
		TransferKey: key.Bytes(),
		TransferMac: []byte("transferMac"),
		NumParts:    16,
		Size:        256,
		Retry:       1.5,
		Preview:     []byte("filePreview"),
	}

	sendMsg, err := newNewFileTransferE2eMessage(recipient, expected.FileName,
		expected.FileType, key, expected.TransferMac, uint16(expected.NumParts),
		expected.Size, expected.Retry, expected.Preview)
	if err != nil {
		t.Errorf("newNewFileTransferE2eMessage returned an error: %+v", err)
	}

	if sendMsg.MessageType != message.NewFileTransfer {
		t.Errorf("Send message has wrong MessageType."+
			"\nexpected: %d\nreceived: %d", message.NewFileTransfer,
			sendMsg.MessageType)
	}

	if !sendMsg.Recipient.Cmp(recipient) {
		t.Errorf("Send message has wrong Recipient."+
			"\nexpected: %s\nreceived: %s", recipient, sendMsg.Recipient)
	}

	received := &NewFileTransfer{}
	err = proto.Unmarshal(sendMsg.Payload, received)
	if err != nil {
		t.Errorf("Failed to unmarshal received NewFileTransfer: %+v", err)
	}

	if !proto.Equal(expected, received) {
		t.Errorf("Received NewFileTransfer does not match expected."+
			"\nexpected: %+v\nreceived: %+v", expected, received)
	}
}
