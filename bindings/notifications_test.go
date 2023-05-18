////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"encoding/json"
	"fmt"
	"testing"

	"gitlab.com/elixxir/client/v4/e2e/ratchet"
	"gitlab.com/xx_network/primitives/id"
)

func TestNotificationReport(t *testing.T) {
	var reports []NotificationReport

	for i := 0; i < 3; i++ {
		nr := NotificationReport{
			ForMe:  true,
			Type:   []string{ratchet.E2e},
			Source: id.NewIdFromUInt(uint64(i), id.User, t).Bytes(),
		}

		reports = append(reports, nr)
	}

	nrs := NotificationReports(reports)

	marshal, _ := json.MarshalIndent(nrs, " ", "	")
	fmt.Printf("%s\n", marshal)
}
