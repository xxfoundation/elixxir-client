////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package receptionID

import (
	"math"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"time"

	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
)

// Tests that generateFakeIdentity is consistent and returns a correct result.
func Test_generateFakeIdentity(t *testing.T) {
	rng := rand.New(rand.NewSource(42))

	expected := IdentityUse{
		Identity: Identity{
			EphemeralIdentity: EphemeralIdentity{
				EphId: ephemeral.Id{0, 0, 0, 0, 0, 0, 46, 197},
				Source: id.NewIdFromBase64String(
					"U4x/lrFkvxuXu59LtHLon1sUhPJSCcnZND6SugndnVID", id.User, t),
			},
			AddressSize: 15,
			End:         time.Unix(0, 1258494203759765625),
			ExtraChecks: 0,
			StartValid:  time.Unix(0, 1258407803759765625),
			EndValid:    time.Unix(0, 1258494203759765625),
			Ephemeral:   true,
			ProcessNext: nil,
		},
		Fake: true,
		UR:   nil,
		ER:   nil,
		CR:   nil,
	}

	timestamp := time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)

	received, err := generateFakeIdentity(rng, expected.AddressSize, timestamp)
	if err != nil {
		t.Errorf("Error generating fake identity: %+v", err)
	}

	if !reflect.DeepEqual(expected, received) {
		t.Errorf("The fake identity was generated incorrectly."+
			"\nexpected: %#v\nreceived: %#v", expected, received)
	}
}

// Error path: generateFakeIdentity fails to generate random bytes.
func Test_generateFakeIdentity_RngError(t *testing.T) {
	rng := strings.NewReader("")
	timestamp := time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)
	expectedErr := "failed to generate a random identity when none is available"

	_, err := generateFakeIdentity(rng, 15, timestamp)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("generateFakeIdentity did not return the correct error on "+
			"failure to generate random bytes.\nexpected: %s\nreceived: %+v",
			expectedErr, err)
	}
}

// Error path: generateFakeIdentity fails to get the address ID.
func Test_generateFakeIdentity_GetEphemeralIdError(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	timestamp := time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)
	expectedErr := "failed to generate an address ID for random identity " +
		"when none is available"

	_, err := generateFakeIdentity(rng, math.MaxInt8, timestamp)
	if err == nil || !strings.Contains(err.Error(), expectedErr) {
		t.Errorf("generateFakeIdentity did not return the correct error on "+
			"failure to generate address ID.\nexpected: %s\nreceived: %+v",
			expectedErr, err)
	}
}
