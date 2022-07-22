///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"encoding/json"
	"testing"
	"time"

	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
)

func TestMessage_Json(t *testing.T) {
	rng := csprng.NewSystemRNG()
	messageID := e2e.MessageID{}
	_, _ = rng.Read(messageID[:])
	payload := make([]byte, 64)
	_, _ = rng.Read(payload)
	sender := id.NewIdFromString("zezima", id.User, t)
	receiver := id.NewIdFromString("jakexx360", id.User, t)
	m := Message{
		MessageType: 1,
		ID:          messageID[:],
		Payload:     payload,
		Sender:      sender.Marshal(),
		RecipientID: receiver.Marshal(),
		EphemeralID: 17,
		Timestamp:   time.Now().UnixNano(),
		Encrypted:   false,
		RoundId:     19,
	}
	mm, _ := json.Marshal(m)
	t.Log("Marshalled Message")
	t.Log(string(mm))
}
