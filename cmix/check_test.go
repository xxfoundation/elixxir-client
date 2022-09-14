////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package cmix

import (
	"testing"

	bloom "gitlab.com/elixxir/bloomfilter"
	"gitlab.com/elixxir/client/cmix/identity/receptionID/store"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/id"
)

// TestChecker tests the basic operation for Checker
func TestChecker(t *testing.T) {
	// RID for testing
	rid := id.Round(2)

	// Init bloom ring buff
	br, err := bloom.Init(10, 0.01)
	if err != nil {
		t.Errorf("Failed to init bloom ring: %+v", err)
	}

	// Create filters object not in range
	filters := []*RemoteFilter{
		{
			data: &mixmessages.ClientBloom{
				Filter:     nil,
				FirstRound: 0,
				RoundRange: 1,
			},
			filter: br,
		},
	}

	// Init a kv and a checked rounds structure
	kv := versioned.NewKV(ekv.MakeMemstore())
	cr, err := store.NewCheckedRounds(5, kv)
	if err != nil {
		t.Errorf("Failed to create checked rounds store: %+v", err)
	}

	ok := Checker(rid, filters, cr)
	if ok {
		t.Errorf("Should not have received OK response when appropriate data not added to stores")
	}

	cr2, err := store.NewCheckedRounds(5, kv)
	if err != nil {
		t.Errorf("Failed to create checked rounds store: %+v", err)
	}
	cr2.Check(rid)
	ok = Checker(rid, filters, cr2)
	if !ok {
		t.Errorf("Checker should have returned ok if round checked in checkrounds")
	}

	br.Add(serializeRound(rid))
	filters = []*RemoteFilter{
		{
			data: &mixmessages.ClientBloom{
				Filter:     nil,
				FirstRound: 1,
				RoundRange: 5,
			},
			filter: br,
		},
	}
	ok = Checker(rid, filters, cr)
	if !ok {
		t.Errorf("Checker did not return OK for round in filter")
	}

}
