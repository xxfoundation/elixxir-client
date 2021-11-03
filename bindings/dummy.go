////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"gitlab.com/elixxir/client/dummy"
	"time"
)

// StartDummyTraffic starts sending dummy traffic. The maxNumMessages is the
// upper bound of the random number of messages sent each send. avgSendDeltaMS
// is the average duration, in milliseconds, to wait between sends. Sends occur
// every avgSendDeltaMS +/- a random duration with an upper bound of
// randomRangeMS.
func StartDummyTraffic(client *Client, maxNumMessages, avgSendDeltaMS,
	randomRangeMS int) error {
	avgSendDelta := time.Duration(avgSendDeltaMS) * time.Millisecond
	randomRange := time.Duration(randomRangeMS) * time.Millisecond

	m := dummy.NewManager(
		maxNumMessages, avgSendDelta, randomRange, &client.api)

	return client.api.AddService(m.StartDummyTraffic)
}
