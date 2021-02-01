package reception

import (
	"encoding/json"
	"math/rand"
	"testing"
	"time"
)

//tests Generate Fake identity is consistant and returns a correct result
func TestGenerateFakeIdentity(t *testing.T) {
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
		"\"EndRequest\":\"0001-01-01T00:00:00Z\",\"Fake\":true,\"KR\":null}"

	timestamp := time.Date(2009, 11, 17, 20,
		34, 58, 651387237, time.UTC)

	received, err := generateFakeIdentity(rng, 15, timestamp)
	if err != nil {
		t.Errorf("Generate fake identity returned an unexpected "+
			"error: %+v", err)
	}

	receivedJson, _ := json.Marshal(received)

	if expected != string(receivedJson) {
		t.Errorf("The fake identity was generated incorrectly\n "+
			"Expected: %s,\n Received: %s", expected, receivedJson)
	}
}
