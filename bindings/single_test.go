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

	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
)

func TestSingleUseJsonMarshals(t *testing.T) {
	rids := []id.Round{1, 5, 9}
	rl := makeRoundsList(rids...)
	rid := id.NewIdFromString("zezima", id.User, t)
	eid, _, _, err := ephemeral.GetId(rid, 16, time.Now().UnixNano())
	if err != nil {
		t.Fatalf("Failed to generate ephemeral ID: %+v", err)
	}
	ephId := receptionID.EphemeralIdentity{
		EphId:  eid,
		Source: rid,
	}
	payload := make([]byte, 64)
	rng := csprng.NewSystemRNG()
	_, _ = rng.Read(payload)
	sendReport := SingleUseSendReport{
		RoundsList:  rl,
		EphID:       ephId.EphId.Int64(),
		ReceptionID: ephId.Source.Marshal(),
	}
	srm, err := json.Marshal(sendReport)
	if err != nil {
		t.Errorf("Failed to marshal send report to JSON: %+v", err)
	} else {
		t.Logf("Marshalled send report:\n%s\n", string(srm))
	}

	responseReport := SingleUseResponseReport{
		RoundsList:  rl,
		Payload:     payload,
		ReceptionID: ephId.Source.Marshal(),
		EphID:       ephId.EphId.Int64(),
		Err:         nil,
	}
	rrm, err := json.Marshal(responseReport)
	if err != nil {
		t.Errorf("Failed to marshal response report to JSON: %+v", err)
	} else {
		t.Logf("Marshalled response report:\n%s\n", string(rrm))
	}

	callbackReport := SingleUseCallbackReport{
		RoundsList:  rl,
		Payload:     payload,
		Partner:     rid,
		EphID:       ephId.EphId.Int64(),
		ReceptionID: ephId.Source.Marshal(),
	}
	crm, err := json.Marshal(callbackReport)
	if err != nil {
		t.Errorf("Failed to marshal callback report to JSON: %+v", err)
	} else {
		t.Logf("Marshalled callback report:\n%s\n", string(crm))
	}
}
