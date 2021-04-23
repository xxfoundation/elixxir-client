package reception

import (
	"encoding/json"
	"math"
	"math/rand"
	"strings"
	"testing"
	"time"
)

// Tests Generate Fake identity is consistent and returns a correct result.
func Test_generateFakeIdentity(t *testing.T) {
	rng := rand.New(rand.NewSource(42))

	end, _ := json.Marshal(time.Unix(0, 1258494203759765625))
	startValid, _ := json.Marshal(time.Unix(0, 1258407803759765625))
	endValid, _ := json.Marshal(time.Unix(0, 1258494203759765625))
	expected := "{\"EphId\":[0,0,0,0,0,0,46,197]," +
		"\"Source\":[83,140,127,150,177,100,191,27,151,187,159,75,180,114," +
		"232,159,91,20,132,242,82,9,201,217,52,62,146,186,9,221,157,82,3]," +
		"\"End\":" + string(end) + ",\"ExtraChecks\":0," +
		"\"StartValid\":" + string(startValid) + "," +
		"\"EndValid\":" + string(endValid) + "," +
		"\"RequestMask\":86400000000000,\"Ephemeral\":true," +
		"\"StartRequest\":\"0001-01-01T00:00:00Z\"," +
		"\"EndRequest\":\"0001-01-01T00:00:00Z\",\"Fake\":true,\"UR\":null,\"ER\":null,\"CR\":null}"

	timestamp := time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)

	received, err := generateFakeIdentity(rng, 15, timestamp)
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

// Error path: fails to get the ephemeral ID.
func Test_generateFakeIdentity_GetEphemeralIdError(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	timestamp := time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)

	_, err := generateFakeIdentity(rng, math.MaxUint64, timestamp)
	if err == nil || !strings.Contains(err.Error(), "ephemeral ID") {
		t.Errorf("generateFakeIdentity() did not return the correct error on "+
			"failure to generate ephemeral ID: %+v", err)
	}
}
