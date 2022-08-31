package bindings

import (
	"encoding/json"
	"fmt"
	"gitlab.com/elixxir/client/e2e/ratchet"
	"gitlab.com/xx_network/primitives/id"
	"testing"
)

func TestNotificationReport(t *testing.T) {
	reports := []NotificationReport{}

	for i := 0; i < 3; i++ {
		nr := NotificationReport{
			ForMe:  true,
			Type:   ratchet.E2e,
			Source: id.NewIdFromUInt(uint64(i), id.User, t).Bytes(),
		}

		reports = append(reports, nr)
	}

	nrs := NotificationReports(reports)

	marshal, _ := json.Marshal(nrs)
	fmt.Printf("%s\n", marshal)
}
