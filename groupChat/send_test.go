////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package groupChat

import (
	"bytes"
	"encoding/base64"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	gs "gitlab.com/elixxir/client/groupChat/groupStore"
	"gitlab.com/elixxir/crypto/group"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/netTime"
	"math/rand"
	"strings"
	"testing"
	"time"
)

func Test_manager_Send(t *testing.T) {
	msgChan := make(chan MessageReceive, 10)

	prng := rand.New(rand.NewSource(42))
	m, g := newTestManagerWithStore(prng, 1, 0, nil, t)
	messageBytes := []byte("Group chat message.")
	reception := &receptionProcessor{
		m: m,
		g: g,
		p: &testProcessor{msgChan},
	}

	roundId, _, msgId, err := m.Send(g.ID, "", messageBytes)
	if err != nil {
		t.Errorf("Send returned an error: %+v", err)
	}

	// Get messages sent with or return an error if no messages were sent
	var messages []format.Message
	if len(m.getCMix().(*testNetworkManager).receptionMessages) > 0 {
		messages = m.getCMix().(*testNetworkManager).receptionMessages[0]
	} else {
		t.Error("No group cMix messages received.")
	}

	timestamps := make(map[states.Round]time.Time)
	timestamps[states.PRECOMPUTING] = netTime.Now().Round(0)
	for _, msg := range messages {
		reception.Process(msg, receptionID.EphemeralIdentity{
			EphId: ephemeral.Id{1, 2, 3}, Source: &id.ID{4, 5, 6},
		},
			rounds.Round{ID: roundId, Timestamps: timestamps})
		select {
		case result := <-msgChan:
			if !result.SenderID.Cmp(m.getReceptionIdentity().ID) {
				t.Errorf("Sender mismatch")
			}
			if result.ID.String() != msgId.String() {
				t.Errorf("MsgId mismatch")
			}
			if !bytes.Equal(result.Payload, messageBytes) {
				t.Errorf("Payload mismatch")
			}
		}
	}
}

// Error path: reader returns an error.
func TestGroup_newCmixMsg_SaltReaderError(t *testing.T) {
	expectedErr := strings.SplitN(saltReadErr, "%", 2)[0]
	m, _ := newTestManager(t)

	_, err := newCmixMsg(
		gs.Group{ID: id.NewIdFromString("test", id.User, t)},
		"", time.Time{}, group.Member{}, strings.NewReader(""),
		m.getCMix().GetMaxMessageLength(), []byte("internal Message"))
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("newCmixMsg failed to return the expected error"+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Error path: size of message is too large for the internalMsg.
func TestGroup_newCmixMsg_InternalMsgSizeError(t *testing.T) {
	expectedErr := strings.SplitN(messageLenErr, "%", 2)[0]

	// Create new test manager and Group
	prng := rand.New(rand.NewSource(42))
	m, g := newTestManagerWithStore(prng, 10, 0, nil, t)

	// Create test parameters
	testMsg := make([]byte, 1500)

	// Create cMix message
	prng = rand.New(rand.NewSource(42))
	_, _, err := m.newMessages(g, "", testMsg, netTime.Now())
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("newCmixMsg failed to return the expected error"+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Tests the consistency of newSalt.
func Test_newSalt_Consistency(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	expectedSalts := []string{
		"U4x/lrFkvxuXu59LtHLon1sUhPJSCcnZND6SugndnVI=",
		"39ebTXZCm2F6DJ+fDTulWwzA1hRMiIU1hBrL4HCbB1g=",
		"CD9h03W8ArQd9PkZKeGP2p5vguVOdI6B555LvW/jTNw=",
		"uoQ+6NY+jE/+HOvqVG2PrBPdGqwEzi6ih3xVec+ix44=",
		"GwuvrogbgqdREIpC7TyQPKpDRlp4YgYWl4rtDOPGxPM=",
		"rnvD4ElbVxL+/b4MECiH4QDazS2IX2kstgfaAKEcHHA=",
		"ceeWotwtwlpbdLLhKXBeJz8FySMmgo4rBW44F2WOEGE=",
		"SYlH/fNEQQ7UwRYCP6jjV2tv7Sf/iXS6wMr9mtBWkrE=",
		"NhnnOJZN/ceejVNDc2Yc/WbXT+weG4lJGrcjbkt1IWI=",
	}

	for i, expected := range expectedSalts {
		salt, err := newSalt(prng)
		if err != nil {
			t.Errorf("newSalt returned an error (%d): %+v", i, err)
		}

		saltString := base64.StdEncoding.EncodeToString(salt[:])

		if expected != saltString {
			t.Errorf("newSalt did not return the expected salt (%d)."+
				"\nexpected: %s\nreceived: %s", i, expected, saltString)
		}

		// fmt.Printf("\"%s\",\n", saltString)
	}
}

// Error path: reader returns an error.
func Test_newSalt_ReadError(t *testing.T) {
	expectedErr := strings.SplitN(saltReadErr, "%", 2)[0]

	_, err := newSalt(strings.NewReader(""))
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("newSalt failed to return the expected error"+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Error path: reader fails to return enough bytes.
func Test_newSalt_ReadLengthError(t *testing.T) {
	expectedErr := strings.SplitN(saltReadLengthErr, "%", 2)[0]

	_, err := newSalt(strings.NewReader("A"))
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("newSalt failed to return the expected error"+
			"\nexpected: %s\nreceived: %+v", expectedErr, err)
	}
}

// Tests that the marshaled internalMsg can be unmarshaled and has all the
// original values.
func Test_setInternalPayload(t *testing.T) {
	internalMessage, err := newInternalMsg(internalMinLen * 2)
	if err != nil {
		t.Errorf("Failed to create a new internalMsg: %+v", err)
	}

	timestamp := netTime.Now()
	sender := id.NewIdFromString("sender ID", id.User, t)
	testMsg := []byte("This is an internal message.")

	payload := setInternalPayload(internalMessage, timestamp, sender, testMsg)
	if err != nil {
		t.Errorf("setInternalPayload returned an error: %+v", err)
	}

	// Attempt to unmarshal and check all values
	unmarshalled, err := unmarshalInternalMsg(payload)
	if err != nil {
		t.Errorf("Failed to unmarshal internalMsg: %+v", err)
	}

	if !timestamp.Equal(unmarshalled.GetTimestamp()) {
		t.Errorf("Timestamp does not match original.\nexpected: %s\nreceived: %s",
			timestamp, unmarshalled.GetTimestamp())
	}

	testSender, err := unmarshalled.GetSenderID()
	if err != nil {
		t.Errorf("Failed to get sender ID: %+v", err)
	}
	if !sender.Cmp(testSender) {
		t.Errorf("Sender ID does not match original.\nexpected: %s\nreceived: %s",
			sender, testSender)
	}

	if !bytes.Equal(testMsg, unmarshalled.GetPayload()) {
		t.Errorf("Payload does not match original.\nexpected: %v\nreceived: %v",
			testMsg, unmarshalled.GetPayload())
	}
}

// Tests that the marshaled publicMsg can be unmarshaled and has all the
// original values.
func Test_setPublicPayload(t *testing.T) {
	prng := rand.New(rand.NewSource(42))
	publicMessage, err := newPublicMsg(publicMinLen * 2)
	if err != nil {
		t.Errorf("Failed to create a new publicMsg: %+v", err)
	}

	var salt [group.SaltLen]byte
	prng.Read(salt[:])
	encryptedPayload := make([]byte, publicMessage.GetPayloadSize())
	copy(encryptedPayload, "This is an internal message.")

	payload := setPublicPayload(publicMessage, salt, encryptedPayload)
	if err != nil {
		t.Errorf("setPublicPayload returned an error: %+v", err)
	}

	// Attempt to unmarshal and check all values
	unmarshalled, err := unmarshalPublicMsg(payload)
	if err != nil {
		t.Errorf("Failed to unmarshal publicMsg: %+v", err)
	}

	if salt != unmarshalled.GetSalt() {
		t.Errorf("Salt does not match original.\nexpected: %v\nreceived: %v",
			salt, unmarshalled.GetSalt())
	}

	if !bytes.Equal(encryptedPayload, unmarshalled.GetPayload()) {
		t.Errorf("Payload does not match original.\nexpected: %v\nreceived: %v",
			encryptedPayload, unmarshalled.GetPayload())
	}
}

type testProcessor struct {
	msgChan chan MessageReceive
}

func (tp *testProcessor) Process(decryptedMsg MessageReceive, _ format.Message,
	_ receptionID.EphemeralIdentity, _ rounds.Round) {
	tp.msgChan <- decryptedMsg
}

func (tp *testProcessor) String() string { return "testProcessor" }
