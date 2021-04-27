package bindings

import (
	"encoding/base64"
	"gitlab.com/elixxir/crypto/fingerprint"
	"gitlab.com/xx_network/primitives/id"
	"testing"
)

func TestNotificationForMe(t *testing.T) {
	payload := []byte("I'm a payload")
	hash := fingerprint.GetMessageHash(payload)
	rid := id.NewIdFromString("zezima", id.User, t)
	fp := fingerprint.IdentityFP(payload, rid)

	ok, err := NotificationForMe(base64.StdEncoding.EncodeToString(hash), base64.StdEncoding.EncodeToString(fp), rid.Bytes())
	if err != nil {
		t.Errorf("Failed to check notification: %+v", err)
	}
	if !ok {
		t.Error("Should have gotten ok response")
	}
}
