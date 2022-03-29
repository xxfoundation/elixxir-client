package receptionID

import (
	"encoding/json"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Tests that generateFakeIdentity is consistent and returns a correct result.
func Test_generateFakeIdentity(t *testing.T) {
	rng := rand.New(rand.NewSource(42))

	addressSize := uint8(15)
	end, _ := json.Marshal(time.Unix(0, 1258494203759765625))
	startValid, _ := json.Marshal(time.Unix(0, 1258407803759765625))
	endValid, _ := json.Marshal(time.Unix(0, 1258494203759765625))
	expected := `{"EphId":[0,0,0,0,0,0,46,197],` +
		`"Source":"U4x/lrFkvxuXu59LtHLon1sUhPJSCcnZND6SugndnVID",` +
		`"AddressSize":` + strconv.Itoa(int(addressSize)) + `,` +
		`"End":` + string(end) + `,"ExtraChecks":0,` +
		`"StartValid":` + string(startValid) + `,` +
		`"EndValid":` + string(endValid) + `,` +
		`"Ephemeral":true,` +
		`"Fake":true,"UR":null,"ER":null,"CR":null}`

	timestamp := time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)

	received, err := generateFakeIdentity(rng, addressSize, timestamp)
	if err != nil {
		t.Errorf("generateFakeIdentity() returned an error: %+v", err)
	}

	receivedJson, _ := json.Marshal(received)

	if expected != string(receivedJson) {
		t.Errorf("The fake identity was generated incorrectly."+
			"\nexpected: %s\nreceived: %s", expected, receivedJson)
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
