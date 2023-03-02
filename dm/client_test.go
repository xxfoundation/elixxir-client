////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package dm

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/crypto/codename"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/nike/ecdh"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/csprng"
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
