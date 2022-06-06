package bindings

import (
	"encoding/json"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"reflect"
	"testing"
	"time"
)

func TestE2ESendReport_JSON(t *testing.T) {
	rng := csprng.NewSystemRNG()
	mid := e2e.MessageID{}
	rng.Read(mid[:])
	origRL := []id.Round{1, 5, 9}
	rl := makeRoundsList(origRL)
	mrl, _ := json.Marshal(&rl)
	sr := E2ESendReport{
		RoundsList: rl,
		MessageID:  mid[:],
		Timestamp:  time.Now().UnixNano(),
	}
	srm, _ := json.Marshal(&sr)
	t.Log("Marshalled RoundsList")
	t.Log(string(mrl))
	t.Log("Marshalled E2ESendReport")
	t.Log(string(srm))
	unmarshalled, err := unmarshalRoundsList(srm)
	if err != nil {
		t.Errorf("Failed to unmarshal rounds list from e2esendreport: %+v", err)
	}
	if !reflect.DeepEqual(unmarshalled, origRL) {
		t.Errorf("Did not receive expected rounds list\n\tExpected: %+v\n\tReceived: %+v\n", rl.Rounds, unmarshalled)
	}
}
