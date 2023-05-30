////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package dm

import (
	"crypto/ed25519"
	"encoding/base64"
	"github.com/stretchr/testify/require"
	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/collective"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/crypto/codename"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/nike/ecdh"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/netTime"
	"math/rand"
	"testing"
)

// TestNick runs basic smoke testing of nick name manager
func TestNick(t *testing.T) {
	crng := fastRNG.NewStreamGenerator(100, 5, csprng.NewSystemRNG)
	rng := crng.GetStream()
	me, _ := codename.GenerateIdentity(rng)
	// partner, _ := codename.GenerateIdentity(rng)
	rng.Close()

	// NOTE: the ID's were lobotomized in the middle of the DM
	// development, s.t. there is only one nick name for everyone
	// right now. Adding nicks per user is a future feature, which
	// is why this test is the way it is and why the API
	// is wonky. For now we are locking in expected behavior but
	// expect to change this in the future.
	myPubKey := ecdh.Edwards2ECDHNIKEPublicKey(&me.PubKey)
	myID := deriveReceptionID(myPubKey.Bytes(),
		me.GetDMToken())

	// partnerPubKey := ecdh.Edwards2ECDHNIKEPublicKey(&partner.PubKey)
	// partnerID := deriveReceptionID(partnerPubKey.Bytes(),
	// 	partner.GetDMToken())

	kv := versioned.NewKV(ekv.MakeMemstore())

	nnm := NewNicknameManager(myID, kv)

	_, ok := nnm.GetNickname()
	require.False(t, ok)

	expectedName := "testuser"

	nnm.SetNickname(expectedName)

	name, ok := nnm.GetNickname()
	require.True(t, ok)
	require.Equal(t, name, expectedName)
	name2, ok := nnm.GetNickname()
	require.True(t, ok)
	require.Equal(t, name2, expectedName)
}

func TestBlock(t *testing.T) {
	netA, netB := createLinkedNets()

	crng := fastRNG.NewStreamGenerator(100, 5, csprng.NewSystemRNG)
	rng := crng.GetStream()
	me, _ := codename.GenerateIdentity(rng)
	partner, _ := codename.GenerateIdentity(rng)
	rng.Close()

	ekvA := collective.TestingKV(t, ekv.MakeMemstore(),
		collective.StandardPrefexs, collective.NewMockRemote())
	ekvB := collective.TestingKV(t, ekv.MakeMemstore(),
		collective.StandardPrefexs, collective.NewMockRemote())

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

	clientA, err := NewDMClient(&me, receiverA, stA, nnmA, netA, ekvA, crng)
	require.NoError(t, err)
	clientB, err := NewDMClient(&partner, receiverB, stB, nnmB, netB, ekvA, crng)
	require.NoError(t, err)

	// make sure the blocklist is empty
	beEmpty := clientB.GetBlockedSenders()
	require.Equal(t, len(beEmpty), 0)

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
	require.Equal(t, 3, len(receiverB.Msgs))

	// User B Blocks User A
	receiverB.BlockSender(rcvA1.PubKey)

	// React to the reply
	pubKey = rcvB1.PubKey
	dmToken = rcvB1.DMToken
	clientA.SendReaction(&pubKey, dmToken, "😀", replyTo2, params)

	// Make sure nothing changed under the hood because the
	// message was dropped.
	require.Equal(t, 3, len(receiverB.Msgs))

	require.True(t, clientB.IsBlocked(clientA.GetIdentity().PubKey))

	// Ensure that this user appears in the blocked senders list:
	blocked := clientB.GetBlockedSenders()
	require.Equal(t, len(blocked), 1)
	require.Equal(t, blocked[0], rcvA1.PubKey)

	// User B Stops blocking User A
	receiverB.UnblockSender(rcvA1.PubKey)

	// React to the reply
	pubKey = rcvB1.PubKey
	dmToken = rcvB1.DMToken
	clientA.SendReaction(&pubKey, dmToken, "😀", replyTo2, params)

	// Make sure reaction is received
	require.Equal(t, 4, len(receiverB.Msgs))
	rcvA2 := receiverB.Msgs[3]
	require.Equal(t, replyTo2, rcvA2.ReplyTo)
	require.Equal(t, "😀", rcvA2.Message)
	require.False(t, clientB.IsBlocked(clientA.GetIdentity().PubKey))

	require.Equal(t, len(clientB.GetBlockedSenders()), 0)

}

func TestDm_mapUpdate(t *testing.T) {

	numEdits := 100

	expectedUpdates := make(map[string]bool, numEdits)
	edits := make(map[string]versioned.ElementEdit, numEdits)

	rng := rand.New(rand.NewSource(69))

	netA, _ := createLinkedNets()

	crng := fastRNG.NewStreamGenerator(100, 5, csprng.NewSystemRNG)
	me, _ := codename.GenerateIdentity(rng)
	partner, _ := codename.GenerateIdentity(rng)

	ekvA := collective.TestingKV(t, ekv.MakeMemstore(),
		collective.StandardPrefexs, collective.NewMockRemote())
	ekvB := collective.TestingKV(t, ekv.MakeMemstore(),
		collective.StandardPrefexs, collective.NewMockRemote())

	stA := NewSendTracker(ekvA)

	receiverA := newMockReceiver()

	myID := DeriveReceptionID(me.PubKey, me.GetDMToken())
	partnerID := deriveReceptionID(partner.PubKey, partner.GetDMToken())

	nnmA := NewNicknameManager(myID, ekvA)
	nnmB := NewNicknameManager(partnerID, ekvB)

	nnmA.SetNickname("me")
	nnmB.SetNickname("partner")

	clientA, err := newDmClient(&me, receiverA, stA, nnmA, netA, ekvA, crng)
	require.NoError(t, err)

	for i := 0; i < numEdits; i++ {
		pubKey, _, err := ed25519.GenerateKey(rng)
		require.NoError(t, err)

		// make 1/3 chance it will be deleted
		existsChoice := make([]byte, 1)
		rng.Read(existsChoice)
		op := versioned.KeyOperation(int(existsChoice[0]) % 3)
		expected := true
		data := base64.StdEncoding.EncodeToString(pubKey)
		if op == versioned.Deleted {
			expected = false
		}

		expectedUpdates[data] = expected
		edits[data] = versioned.ElementEdit{
			OldElement: nil,
			NewElement: &versioned.Object{
				Version:   0,
				Timestamp: netTime.Now(),
				Data:      pubKey,
			},
			Operation: op,
		}
	}

	clientA.mapUpdate(dmMapName, edits)

	for key, shouldBeBlocked := range expectedUpdates {
		pubKey, err := base64.StdEncoding.DecodeString(key)
		require.NoError(t, err)
		if !shouldBeBlocked {
			require.False(t, clientA.IsBlocked(pubKey))
		} else {
			require.True(t, clientA.IsBlocked(pubKey))
		}

	}
}
