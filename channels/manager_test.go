package channels

import (
	"os"
	"testing"

	jww "github.com/spf13/jwalterweatherman"
)

func TestMain(m *testing.M) {
	// Many tests trigger WARN prints;, set the out threshold so the WARN prints
	// can be seen in the logs
	jww.SetStdoutThreshold(jww.LevelWarn)
	os.Exit(m.Run())
}
