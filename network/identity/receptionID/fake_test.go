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

// Tests Generate Fake identity is consistent and returns a correct result.
func Test_generateFakeIdentity(t *testing.T) {
	rng := rand.New(rand.NewSource(42))

	addressSize := uint8(15)
	end, _ := json.Marshal(time.Unix(0, 1258494203759765625))
	startValid, _ := json.Marshal(time.Unix(0, 1258407803759765625))
	endValid, _ := json.Marshal(time.Unix(0, 1258494203759765625))
	expected := "{\"EphId\":[0,0,0,0,0,0,46,197]," +
		"\"Source\":\"U4x/lrFkvxuXu59LtHLon1sUhPJSCcnZND6SugndnVID\"," +
		"\"AddressSize\":" + strconv.Itoa(int(addressSize)) + "," +
		"\"End\":" + string(end) + ",\"ExtraChecks\":0," +
		"\"StartValid\":" + string(startValid) + "," +
		"\"EndValid\":" + string(endValid) + "," +
		"\"Ephemeral\":true," +
		"\"Fake\":true,\"UR\":null,\"ER\":null,\"CR\":null}"

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

// Error path: fails to generate random bytes.
func Test_generateFakeIdentity_RngError(t *testing.T) {
	rng := strings.NewReader("")
	timestamp := time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)

	_, err := generateFakeIdentity(rng, 15, timestamp)
	if err == nil || !strings.Contains(err.Error(), "failed to generate a random identity") {
		t.Errorf("generateFakeIdentity() did not return the correct error on "+
			"failure to generate random bytes: %+v", err)
	}
}

// Error path: fails to get the address ID.
func Test_generateFakeIdentity_GetEphemeralIdError(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	timestamp := time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)

	_, err := generateFakeIdentity(rng, math.MaxInt8, timestamp)
	if err == nil || !strings.Contains(err.Error(), "address ID") {
		t.Errorf("generateFakeIdentity() did not return the correct error on "+
			"failure to generate address ID: %+v", err)
	}
}
