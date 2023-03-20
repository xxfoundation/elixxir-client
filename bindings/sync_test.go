package bindings

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"gitlab.com/xx_network/primitives/netTime"
	"testing"
	"time"
)

func TestName(t *testing.T) {
	rprt := RemoteStoreReport{
		Key:          "exampleKey",
		Value:        []byte("exampleValue"),
		LastModified: netTime.Now().Add(24 * time.Hour).UnixNano(),
		LastWrite:    netTime.Now().Add(12 * time.Hour).Add(2 * time.Minute).UnixNano(),
		Error:        "Example error (may not exist if successful)",
	}

	dum, err := json.MarshalIndent(rprt, "", "")
	require.NoError(t, err)

	t.Logf("%s", dum)
}
