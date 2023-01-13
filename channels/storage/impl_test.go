////////////////////////////////////////////////////////////////////////////////
// Copyright © 2023 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package storage

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	cryptoBroadcast "gitlab.com/elixxir/crypto/broadcast"
	"gitlab.com/elixxir/crypto/message"
	"gitlab.com/xx_network/primitives/id"
	"testing"
	"time"
)

// Series of interdependent smoke tests of the impl object and its methods.
func TestImpl(t *testing.T) {
	jww.SetStdoutThreshold(jww.LevelDebug)

	model, err := newImpl("", nil)
	if err != nil {
		t.Fatal(err)
	}

	testString := "test"
	testChannelId := &id.DummyUser
	testChannel := &cryptoBroadcast.Channel{
		ReceptionID: testChannelId,
		Name:        testString,
		Description: testString,
	}
	model.JoinChannel(testChannel)

	testBytes := []byte(testString)
	testRoundId := uint64(10)
	testMsgId := message.DeriveChannelMessageID(testChannelId,
		testRoundId, testBytes)
	testRound := rounds.Round{ID: id.Round(testRoundId)}
	newId := model.ReceiveMessage(testChannelId, testMsgId, testString, testString, testBytes,
		0, 0, time.Now(), 0, testRound, 0, 0, false)
	t.Logf("Inserted message with ID: %d", newId)

	gotMsg, err := model.GetMessage(testMsgId)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Got Message: %v", gotMsg)

	//updatedId := model.UpdateFromMessageID()
	//
	//model.LeaveChannel(testChannelId)
}
