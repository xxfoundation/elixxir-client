package bindings

import (
	"encoding/json"
	"gitlab.com/elixxir/crypto/e2e"
	"math/rand"
	"testing"
	"time"
)

func TestName(t *testing.T) {
	rl := []uint64{1, 4, 9}
	prng := rand.New(rand.NewSource(42))
	rfp := make([]byte, 32)
	prng.Read(rfp)
	mid := e2e.NewMessageID(rfp, prng.Uint64())

	randData := make([]byte, 32)
	prng.Read(randData)
	k := e2e.Key{}
	copy(k[:], randData)
	kr := e2e.NewKeyResidue(k)

	report := E2ESendReport{
		RoundsList: RoundsList{rl},
		MessageID:  mid.Marshal(),
		Timestamp:  time.Now().UnixNano(),
		KeyResidue: kr.Marshal(),
	}

	marshal, _ := json.Marshal(report)
	t.Logf("%s", marshal)
}
