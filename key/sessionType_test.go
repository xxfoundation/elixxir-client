package key

import (
	"math"
	"testing"
)

//tests the stringers for all possible sessions types are correct
func TestSessionType_String(t *testing.T) {
	for i := 0; i <= math.MaxUint8; i++ {
		st := SessionType(i)
		if st.String() != correctString(i) {
			t.Errorf("Session Name for %v incorrect. Expected: %s, "+
				"Received: %s", i, correctString(i), st.String())
		}
	}
}

func correctString(i int) string {
	switch i {
	case 0:
		return "Send"
	case 1:
		return "Receive"
	default:
		return "Unknown"
	}
}
