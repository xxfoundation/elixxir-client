////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package dm

import (
	"os"
	"testing"

	jww "github.com/spf13/jwalterweatherman"
	"github.com/stretchr/testify/require"
	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/crypto/codename"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/csprng"
)

// TestMain sets the log level so we see important debug messages
func TestMain(m *testing.M) {
	jww.SetStdoutThreshold(jww.LevelInfo)
	os.Exit(m.Run())
}

// TestE2EDMs does a full End-to-End smoke test of DMs: Sending,
// Replying to what was sent, and reacting to that reply.
//
// NOTE: A lot of the "hard" work is actually done in
// interfaces_test.go which mocks the necessary network and receiver
// interfaces to enable sending and receiving without a cMix
// connection.
func TestE2EDMs(t *testing.T) {
	netA, netB := createLinkedNets()

	crng := fastRNG.NewStreamGenerator(100, 5, csprng.NewSystemRNG)
	rng := crng.GetStream()
	me, _ := codename.GenerateIdentity(rng)
	partner, _ := codename.GenerateIdentity(rng)
	rng.Close()

	ekvA := versioned.NewKV(ekv.MakeMemstore())
	ekvB := versioned.NewKV(ekv.MakeMemstore())

	stA := NewSendTracker(ekvA)
	stB := NewSendTracker(ekvB)

	receiverA := newMockReceiver()
	receiverB := newMockReceiver()

	myID := DeriveReceptionID(me.PubKey, me.GetDMToken())
	partnerID := deriveReceptionID(partner.PubKey, partner.GetDMToken())

	nnmA := NewNicknameManager(myID, ekvA)
	nnmB := NewNicknameManager(partnerID, ekvB)

	nnmA.SetNickname("me")
	nnmB.SetNickname("partner")

	clientA := NewDMClient(&me, receiverA, stA, nnmA, netA, crng)
	clientB := NewDMClient(&partner, receiverB, stB, nnmB, netB, crng)

	params := cmix.GetDefaultCMIXParams()

	// Send and receive a text
	clientA.SendText(&partner.PubKey, partner.GetDMToken(), "Hi", params)
	require.Equal(t, 1, len(receiverB.Msgs))
	rcvA1 := receiverB.Msgs[0]
	require.Equal(t, "Hi", rcvA1.Message)

	// Reply to it
	pubKey := rcvA1.PubKey
	dmToken := rcvA1.DMToken
	replyTo := rcvA1.MessageID
	clientB.SendReply(&pubKey, dmToken, "whatup?", replyTo, params)
	require.Equal(t, 3, len(receiverA.Msgs))
	rcvB1 := receiverA.Msgs[2]
	replyTo2 := rcvB1.ReplyTo
	require.Equal(t, replyTo, replyTo2)
	require.Equal(t, "whatup?", rcvB1.Message)

	// React to the reply
	pubKey = rcvB1.PubKey
	dmToken = rcvB1.DMToken
	clientA.SendReaction(&pubKey, dmToken, "😀", replyTo2, params)
	require.Equal(t, 4, len(receiverB.Msgs))
	rcvA2 := receiverB.Msgs[3]
	require.Equal(t, replyTo2, rcvA2.ReplyTo)
	require.Equal(t, "😀", rcvA2.Message)
}
