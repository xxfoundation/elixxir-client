////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package dummy

import (
	"encoding/base64"
	"testing"
	"time"
)

// Consistency test: tests that randomInt returns the expected int when using a
// PRNG and that the result is not larger than the max.
func Test_intRng_Consistency(t *testing.T) {
	expectedInts := []int{15, 1, 35, 13, 42, 52, 57, 3, 48}

	prng := NewPrng(42)
	max := 64

	for i, expected := range expectedInts {
		v, err := randomInt(max, prng)
		if err != nil {
			t.Errorf("randomInt returned an error (%d): %+v", i, err)
		}

		if v != expected {
			t.Errorf("New int #%d does not match expected."+
				"\nexpected: %d\nreceived: %d", i, expected, v)
		}

		// Ensure that the int is in range
		if v > max || v < 1 {
			t.Errorf("Int #%d not within range."+
				"\nexpected: %d < d < %d\nreceived: %d", i, 0, max, v)
		}
	}
}

// Consistency test: tests that randomDuration returns the expected int when using
// a PRNG and that the result is within the allowed range.
func Test_durationRng_Consistency(t *testing.T) {
	expectedDurations := []time.Duration{
		61460632462, 69300060600, 46066982720, 68493307162, 45820762465,
		56472560211, 68610237306, 45503877311, 63543617747,
	}

	prng := NewPrng(42)
	base, randomRange := time.Minute, 15*time.Second

	for i, expected := range expectedDurations {
		v, err := randomDuration(base, randomRange, prng)
		if err != nil {
			t.Errorf("randomDuration returned an error (%d): %+v", i, err)
		}

		if v != expected {
			t.Errorf("New duration #%d does not match expected."+
				"\nexpected: %s\nreceived: %s", i, expected, v)
		}

		// Ensure that the duration is within range
		if v > base+randomRange || v < base-randomRange {
			t.Errorf("Duration #%d is not in range."+
				"\nexpected: %s < d < %s\nreceived: %s", i, base-randomRange,
				base+randomRange, v)
		}
	}
}

// Consistency test: tests that newRandomPayload returns the expected payload
// when using a PRNG and that the result is not larger than the max payload.
func Test_newRandomPayload_Consistency(t *testing.T) {
	expectedPayloads := []string{
		"l7ufS7Ry6J9bFITyUgnJ",
		"Ut/Xm012Qpthegyfnw07pVsMwNYUTIiFNQ==",
		"CD9h",
		"GSnh",
		"joE=",
		"uoQ+6NY+jE/+HOvqVG2PrBPdGqwEzi6ih3xVec+ix44bC6+uiBuCpw==",
		"qkNGWnhiBhaXiu0M48bE8657w+BJW1cS/v2+DBAoh+EA2s0tiF9pLLYH2gChHBxwcec=",
		"suEpcF4nPwXJIyaCjisFbg==",
		"R/3zREEO1MEWAj+o41drb+0n/4l0usDK/ZrQVpKxNhnnOJZN/ceejVNDc2Yc/WbXTw==",
		"bkt1IQ==",
	}

	prng := NewPrng(42)
	maxPayloadSize := 64

	for i, expected := range expectedPayloads {
		payload, err := newRandomPayload(maxPayloadSize, prng)
		if err != nil {
			t.Errorf("newRandomPayload returned an error (%d): %+v", i, err)
		}

		payloadString := base64.StdEncoding.EncodeToString(payload)

		if payloadString != expected {
			t.Errorf("New payload #%d does not match expected."+
				"\nexpected: %s\nreceived: %s", i, expected, payloadString)
		}

		// Ensure that the payload is not larger than the max size
		if len(payload) > maxPayloadSize {
			t.Errorf("Length of payload #%d longer than max allowed."+
				"\nexpected: <%d\nreceived: %d", i, maxPayloadSize, len(payload))
		}
	}
}

// Consistency test: tests that newRandomFingerprint returns the expected
// fingerprints when using a PRNG. Also tests that the first bit is zero.
func Test_newRandomFingerprint_Consistency(t *testing.T) {
	expectedFingerprints := []string{
		"U4x/lrFkvxuXu59LtHLon1sUhPJSCcnZND6SugndnVI=",
		"X9ebTXZCm2F6DJ+fDTulWwzA1hRMiIU1hBrL4HCbB1g=",
		"CD9h03W8ArQd9PkZKeGP2p5vguVOdI6B555LvW/jTNw=",
		"OoQ+6NY+jE/+HOvqVG2PrBPdGqwEzi6ih3xVec+ix44=",
		"GwuvrogbgqdREIpC7TyQPKpDRlp4YgYWl4rtDOPGxPM=",
		"LnvD4ElbVxL+/b4MECiH4QDazS2IX2kstgfaAKEcHHA=",
		"ceeWotwtwlpbdLLhKXBeJz8FySMmgo4rBW44F2WOEGE=",
		"SYlH/fNEQQ7UwRYCP6jjV2tv7Sf/iXS6wMr9mtBWkrE=",
		"NhnnOJZN/ceejVNDc2Yc/WbXT+weG4lJGrcjbkt1IWI=",
		"EM8r60LDyicyhWDxqsBnzqbov0bUqytGgEAsX7KCDog=",
	}

	prng := NewPrng(42)

	for i, expected := range expectedFingerprints {
		fp, err := newRandomFingerprint(prng)
		if err != nil {
			t.Errorf("newRandomFingerprint returned an error (%d): %+v", i, err)
		}

		if fp.String() != expected {
			t.Errorf("New fingerprint #%d does not match expected."+
				"\nexpected: %s\nreceived: %s", i, expected, fp)
		}

		// Ensure that the first bit is zero
		if fp[0]>>7 != 0 {
			t.Errorf("First bit of fingerprint #%d is not 0."+
				"\nexpected: %d\nreceived: %d", i, 0, fp[0]>>7)
		}
	}
}

// Consistency test: tests that newRandomMAC returns the expected MAC when using
// a PRNG. Also tests that the first bit is zero.
func Test_newRandomMAC_Consistency(t *testing.T) {
	expectedMACs := []string{
		"U4x/lrFkvxuXu59LtHLon1sUhPJSCcnZND6SugndnVI=",
		"X9ebTXZCm2F6DJ+fDTulWwzA1hRMiIU1hBrL4HCbB1g=",
		"CD9h03W8ArQd9PkZKeGP2p5vguVOdI6B555LvW/jTNw=",
		"OoQ+6NY+jE/+HOvqVG2PrBPdGqwEzi6ih3xVec+ix44=",
		"GwuvrogbgqdREIpC7TyQPKpDRlp4YgYWl4rtDOPGxPM=",
		"LnvD4ElbVxL+/b4MECiH4QDazS2IX2kstgfaAKEcHHA=",
		"ceeWotwtwlpbdLLhKXBeJz8FySMmgo4rBW44F2WOEGE=",
		"SYlH/fNEQQ7UwRYCP6jjV2tv7Sf/iXS6wMr9mtBWkrE=",
		"NhnnOJZN/ceejVNDc2Yc/WbXT+weG4lJGrcjbkt1IWI=",
		"EM8r60LDyicyhWDxqsBnzqbov0bUqytGgEAsX7KCDog=",
	}

	prng := NewPrng(42)

	for i, expected := range expectedMACs {
		mac, err := newRandomMAC(prng)
		if err != nil {
			t.Errorf("newRandomMAC returned an error (%d): %+v", i, err)
		}

		macString := base64.StdEncoding.EncodeToString(mac)

		if macString != expected {
			t.Errorf("New MAC #%d does not match expected."+
				"\nexpected: %s\nreceived: %s", i, expected, macString)
		}

		// Ensure that the first bit is zero
		if mac[0]>>7 != 0 {
			t.Errorf("First bit of MAC #%d is not 0."+
				"\nexpected: %d\nreceived: %d", i, 0, mac[0]>>7)
		}
	}
}
