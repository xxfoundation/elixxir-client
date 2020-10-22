package params

import (
	"testing"
	"time"
)

func TestGetDefaultCMIX(t *testing.T) {
	c := GetDefaultCMIX()
	if c.RoundTries != 3 || c.Timeout != 25*time.Second {
		t.Errorf("GetDefaultCMIX did not return expected values")
	}
}
