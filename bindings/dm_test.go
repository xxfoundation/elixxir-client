////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"gitlab.com/elixxir/client/v4/dm"
	"gitlab.com/elixxir/crypto/codename"
	cryptoDM "gitlab.com/elixxir/crypto/dm"
	"gitlab.com/xx_network/primitives/id"
	"io"
	"math/rand"
	"testing"
)

// Produces example JSON of DmNotificationUpdateJSON to be used for
// documentation.
func Test_DmNotificationUpdateJSON(t *testing.T) {
	prng := rand.New(rand.NewSource(38496))
	me, _ := codename.GenerateIdentity(prng)
	receptionID, _ := id.NewRandomID(prng, id.User)

	key1, key2, key3, key4 :=
		newPubKey(prng), newPubKey(prng), newPubKey(prng), newPubKey(prng)
	tag2, tag4 := cryptoDM.MakeReceiverSihTag(key2, me.Privkey),
		cryptoDM.MakeReceiverSihTag(key4, me.Privkey)

	nuJSON := DmNotificationUpdateJSON{
		NotificationFilter: dm.NotificationFilter{
			Identifier:   me.PubKey,
			MyID:         receptionID,
			Tags:         []string{tag2, tag4},
			PublicKeys:   map[string]ed25519.PublicKey{tag2: key2, tag4: key4},
			AllowedTypes: map[dm.MessageType]struct{}{dm.TextType: {}, dm.ReplyType: {}},
		},
		Changed: []dm.NotificationState{{key2, dm.NotifyAll}, {key3, dm.NotifyNone}},
		Deleted: []ed25519.PublicKey{key1},
	}

	data, err := json.MarshalIndent(nuJSON, "//  ", "  ")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("//  %s\n", data)
}

// Produces example JSON of DmBlockedUserJSON to be used for documentation.
func Test_DmBlockedUserJSON(t *testing.T) {
	prng := rand.New(rand.NewSource(65622))

	buJSON := DmBlockedUserJSON{
		User:    newPubKey(prng),
		Blocked: true,
	}

	data, err := json.MarshalIndent(buJSON, "//  ", "  ")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("//  %s\n", data)
}

// Produces example JSON of DmMessageReceivedJSON to be used for documentation.
func Test_DmMessageReceivedJSON(t *testing.T) {
	prng := rand.New(rand.NewSource(623677))

	buJSON := DmMessageReceivedJSON{
		UUID:               prng.Uint64(),
		PubKey:             newPubKey(prng),
		MessageUpdate:      true,
		ConversationUpdate: true,
	}

	data, err := json.MarshalIndent(buJSON, "//  ", "  ")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("//  %s\n", data)
}

func newPubKey(rng io.Reader) ed25519.PublicKey {
	pubKey, _, _ := ed25519.GenerateKey(rng)
	return pubKey
}
