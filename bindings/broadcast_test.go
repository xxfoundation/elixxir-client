package bindings

import (
	"encoding/json"
	"gitlab.com/elixxir/crypto/cmix"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"testing"
	"time"
)

func TestChannelDef_JSON(t *testing.T) {
	rng := csprng.NewSystemRNG()
	rng.SetSeed([]byte("rng"))
	pk, _ := rsa.GenerateKey(rng, 256)
	cd := ChannelDef{
		Name:        "My broadcast channel",
		Description: "A broadcast channel for me to test things",
		Salt:        cmix.NewSalt(rng, 16),
		PubKey:      rsa.CreatePublicKeyPem(pk.GetPublic()),
	}

	cdJson, err := json.Marshal(cd)
	if err != nil {
		t.Errorf("Failed to marshal channel def: %+v", err)
	}
	t.Log(string(cdJson))
}

func TestBroadcastMessage_JSON(t *testing.T) {
	uid := id.NewIdFromString("zezima", id.User, t)
	eid, _, _, err := ephemeral.GetId(uid, 16, time.Now().UnixNano())
	if err != nil {
		t.Errorf("Failed to form ephemeral ID: %+v", err)
	}
	bm := BroadcastMessage{
		BroadcastReport: BroadcastReport{
			RoundsList: makeRoundsList(42),
			EphID:      eid,
		},
		Payload: []byte("Hello, broadcast friends!"),
	}
	bmJson, err := json.Marshal(bm)
	if err != nil {
		t.Errorf("Failed to marshal broadcast message: %+v", err)
	}
	t.Log(string(bmJson))
}

func TestBroadcastReport_JSON(t *testing.T) {
	uid := id.NewIdFromString("zezima", id.User, t)
	eid, _, _, err := ephemeral.GetId(uid, 16, time.Now().UnixNano())
	if err != nil {
		t.Errorf("Failed to form ephemeral ID: %+v", err)
	}
	br := BroadcastReport{
		RoundsList: makeRoundsList(42),
		EphID:      eid,
	}

	brJson, err := json.Marshal(br)
	if err != nil {
		t.Errorf("Failed to marshal broadcast report: %+v", err)
	}
	t.Log(string(brJson))
}
