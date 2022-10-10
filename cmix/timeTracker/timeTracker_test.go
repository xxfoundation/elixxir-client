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

func TestAverage(t *testing.T) {
	t1 := time.Duration(int64(10))
	t2 := time.Duration(int64(20))
	t3 := time.Duration(int64(30))
	t4 := time.Duration(int64(1000))
	durations := make([]*time.Duration, 100)
	durations[0] = &t1
	durations[1] = &t2
	durations[2] = &t3
	durations[3] = &t4
	avg := average(durations)
	require.Equal(t, int(avg), 265)
}

func TestGatewayDelayAverage(t *testing.T) {
	t1 := time.Duration(int64(10))
	t2 := time.Duration(int64(20))
	t3 := time.Duration(int64(30))
	t4 := time.Duration(int64(1000))
	gwDelays := newGatewayDelays()
	gwDelays.Add(t1)
	gwDelays.Add(t2)
	gwDelays.Add(t3)
	gwDelays.Add(t4)
	avg := gwDelays.Average()
	require.Equal(t, int(avg), 265)
}
