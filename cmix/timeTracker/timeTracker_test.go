////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

// package timeTracker tracks local clock skew relative to gateways.
package timeTracker

import (
	"crypto/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"gitlab.com/xx_network/primitives/id"
)

func TestTimeTrackerSmokeTest(t *testing.T) {
	tracker := New()
	gwID := &id.ID{}
	_, err := rand.Read(gwID[:])
	require.NoError(t, err)

	startTime := time.Now().AddDate(0, 0, -1) // this time yesterday
	rTs := startTime.Add(time.Second * 10)
	rtt := time.Second * 10
	gwD := time.Second * 3

	tracker.Add(gwID, startTime, rTs, rtt, gwD)
	tracker.Add(gwID, startTime, rTs, rtt, gwD)
	tracker.Add(gwID, startTime, rTs, rtt, gwD)

	aggregate := tracker.Aggregate()

	t.Logf("aggregate: %v", aggregate)
}
