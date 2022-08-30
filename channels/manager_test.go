package channels

import (
	"os"
	"testing"

	jww "github.com/spf13/jwalterweatherman"
)

func TestMain(m *testing.M) {
	// many tests trigger warn prints, set the out threshold so the warns
	// can be seen in the logs
	jww.SetStdoutThreshold(jww.LevelWarn)
	os.Exit(m.Run())
}
