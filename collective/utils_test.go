package collective

import (
	"encoding/base64"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// dvcOffsetSerialEnc is the hardcoded deviceOffset.
const dvcOffsetSerialEnc = "WFhES1RYTE9HRFZDT0ZGU1RleUl0Y1VNdGFIWlNXR3RvTkNJNk56UXNJakJJTTBZNVYxbGhVbEZGSWpvM05pd2lNVGR3YkVscmVXbGhXR2NpT2pjNExDSXhTME5oVG5ac2MwbEtVU0k2TkRFc0lqRk5SVmRCYWkxdk5ERmpJam95T1N3aU1WaEJWRTg0U1ZaalpsVWlPalkyTENJeGVtYzNiVlI0Y0ZOYWF5STZPRFFzSWpKaFdFOVpkalo0UjI5TklqbzNPU3dpTXpKaFVHZ3dOSE51ZUhjaU9qVTNMQ0l6T1dWaVZGaGFRMjB5UlNJNk5Dd2lORXA1YjFsQ1ptNDBhR01pT2pVNExDSTFMVGhMWW1kSWVHaFZNQ0k2TlRFc0lqVTFOVXgyVjE5cVZFNTNJam94TVN3aU5XcG5hbEZ1V21GRk9Wa2lPall6TENJMk1WOXJTbEEzTTNKRFJTSTZOVElzSWpsVmNHdE5VVlpTVURsbklqbzJOeXdpUVVWR2RuQm9ZVloxVTBFaU9qa3hMQ0pCVG5KT1RGbG9abUZUZHlJNk1qSXNJa0pYTkRSR01sZFBSVWRGSWpveU55d2lRMFE1YURBelZ6aEJjbEVpT2pnc0lrTktTREZzTVRSd1MzRkpJam8xTXl3aVJFMUVWMFpGZVVsb1ZGVWlPallzSWtScVNFTjVWazlYV1ZCQklqbzRPQ3dpUlRrd1lYSkJWRTlNY1VraU9qRTBMQ0pHTUdKcU4xVXliVkp4YXlJNk5ESXNJa1p4U2pCSk9FdGxVa000SWpvNE55d2lSMU15ZHpWemFsaERZbXNpT2pnekxDSkhjbU5xWW10ME1VbFhTU0k2TXpVc0lrZDNkWFp5YjJkaVozRmpJam94Tml3aVNFVnFkRVl4VDJSaFJqZ2lPall3TENKSVpsUTFSMU51YUdvNWJ5STZPU3dpU1ZwbVlUVnlZM2wzTVVVaU9qVTJMQ0pLVDBsM1ZYaEthbVpLYXlJNk56RXNJa3hEY2tkVVlqbEhSbEpuSWpvM01Dd2lURzFuTWs5b2VEUm1NR01pT2preUxDSk5NRU5oWTFOMGNWcHhjeUk2Tnpjc0lrMDFRbHBHVFdwTlNGQkJJam8wT0N3aVRXOVdaemhoY2tGYU9EUWlPak0zTENKT1JEWlRkV2R1Wkc1V1NTSTZNeXdpVG1odWJrOUtXazVmWTJNaU9qTXlMQ0pRVEV4NVdGWk1OUzFOWXlJNk9UWXNJbEIzV0VwSmVXRkRhbWx6SWpveU5pd2lVWEJ6V2xOMVJ6WnVlVzhpT2pZNExDSlNTVEJ6VWpVd2VWbElWU0k2TkRjc0lsSmlRM2Q2TFZCalRITkZJam80TlN3aVUwUkJYemxDZDJKSkxVVWlPalU1TENKVFdXeElYMlpPUlZGUk5DSTZNamdzSWxSb1RXWklibmxDU0Rjd0lqbzVOQ3dpVlRSNFgyeHlSbXQyZUhNaU9qQXNJbFZTUTB0UmRUQTRhMFIzSWpveE55d2lWa2h5V0RSa1dXNXFTamdpT2prM0xDSlhNMU41TkZOc2QxaHBZeUk2TWpVc0lsZDRVMFU0YkVsS2VXUnJJam95TENKWU4wNHRVbU13WWpoTWR5STZPRGtzSWxoVVNtYzRaRFpZWjI5TklqbzBNQ3dpV0dKTFIxTjVjbFIzYlhjaU9qWXlMQ0pZYVcxbk0wdFNjWGMyUVNJNk5qVXNJbG94VldsdVFtYzRSMVJOSWpvNE1Td2lXblJrVURkQ05HSnBWV3NpT2pNMExDSmZRbTh3WWtWT1VFVmpPQ0k2T1RBc0lsOUtXVkV4TFRaNGJsRnJJam80Tml3aVgyVXlkV3M1YzJ0c1dFVWlPams1TENKZmFIcHlObXhTZEdvMmR5STZNVE1zSWw5Mk1pMUVRa0Z2YUMxRklqb3lNU3dpWVRKZmRFcGZMVXBrVEc4aU9qTXdMQ0pqTFhadVRFTmxaMjFOUVNJNk5UVXNJbU5sWlZkdmRIZDBkMnh2SWpveU5Dd2laSFpMYm0xTWVHc3paelFpT2pZeExDSmxaM2xtYm5jd04zQldjeUk2TlN3aVprbGpkbWhCTFdSc1Rtc2lPams0TENKbk1tTmFRbmt4Ymt3eVJTSTZOek1zSW1jMFlYVmpjazV0U0MxbklqbzJPU3dpWjBWQmMxZzNTME5FYjJjaU9qTTVMQ0pvTTNoV1pXTXRhWGcwTkNJNk1UVXNJbWhDY2t3MFNFTmlRakZuSWpvM0xDSm9UME4xTW1ObFJscENaeUk2T1RNc0ltbEdkazFhZWxSdmEzbzRJam81TlN3aWFuaGFhVXh5VmswM2FqUWlPamd3TENKclRUaHlOakJNUkhscFl5STZNellzSW13MGNuUkVUMUJIZUZCTklqb3hPU3dpYkRkMVpsTTNVbmsyU2pnaU9qRXNJbXgxVlhRNU1rUXlkekJaSWpvMk5Dd2libUZQWlVSd1MyUkJhMjhpT2pRNUxDSnViUzFETlZVMU1HcHZSU0k2TVRBc0ltNXZNVlJSTTA1dFNGQXdJam96TXl3aWNIVnBYMUowVTNKTE1Ga2lPak00TENKeFVscFpXR2huTjNSM2F5STZPRElzSW5GclRrZFhibWhwUW1oWklqb3hPQ3dpY201MlJEUkZiR0pXZUVraU9qSXdMQ0p5YjNSUWNEZFVPRTlwUVNJNk5ETXNJblJTU1ZvMmJteFVUM0k0SWpvME5pd2lkR2RtWVVGTFJXTklTRUVpT2pJekxDSjFTMjlGWmtkSVRuTjZUU0k2TlRRc0luVk1SRVJ0ZGtoUU1ISk5Jam8wTlN3aWRXOVJMVFpPV1MxcVJUZ2lPakV5TENKMWRtOWhaR1V3ZVdWdldTSTZORFFzSW5aTWVYaGhWR3czZEhwUklqbzFNQ3dpZHpOM1QxSm5lbmN4ZDAwaU9qYzFMQ0ozVFhJNWJYUkNWMnR5UlNJNk16RXNJbmR5YURsaVprUmtXSFpKSWpvM01uMD0"

// Full test for deviceOffset.
func TestDeviceOffset(t *testing.T) {
	dvcOffset := newDeviceOffset()
	require.Len(t, dvcOffset, 0)
	rng := rand.New(rand.NewSource(42))

	// Populate offset structure with data
	const numTests = 100
	for i := 0; i < numTests; i++ {
		instanceID, _ := NewRandomInstanceID(rng)
		dvcOffset[instanceID] = i
	}

	require.Len(t, dvcOffset, numTests)

	// Serialize device offset
	dvcOffsetSerial, err := dvcOffset.serialize()
	require.NoError(t, err)

	// Check that device offset matches hardcoded value
	require.Equal(t, dvcOffsetSerialEnc, base64.RawStdEncoding.EncodeToString(dvcOffsetSerial))

	// Deserialize offset
	deserial, err := deserializeDeviceOffset(dvcOffsetSerial)
	require.NoError(t, err)

	// Ensure deserialized offset matches original
	require.Equal(t, dvcOffset, deserial)
}

// constructTimestamps is a testing utility function. It constructs a list of
// out-of order mock timestamps. By default, it creates a list of 6 hard-coded
// timestamps. It will also append to that list the number of random timestamps.
func constructTimestamps(t require.TestingT, numRandomTimestamps int) []time.Time {
	var (
		timestamp0, timestamp1, timestamp2, timestamp3, timestamp4,
		timestamp5 time.Time
		err error
	)

	rng := rand.New(rand.NewSource(8675309))

	// Construct timestamps. All of these are the same date but with different
	// years.
	timestamp0, err = time.Parse(time.RFC3339,
		"2015-12-21T22:08:41+00:00")
	require.NoError(t, err)

	timestamp1, err = time.Parse(time.RFC3339,
		"2013-12-21T22:08:41+00:00")
	require.NoError(t, err)

	timestamp2, err = time.Parse(time.RFC3339,
		"2003-12-21T22:08:41+00:00")
	require.NoError(t, err)

	timestamp3, err = time.Parse(time.RFC3339,
		"2012-12-21T22:08:41+00:00")
	require.NoError(t, err)

	timestamp4, err = time.Parse(time.RFC3339,
		"2014-12-21T22:08:41+00:00")
	require.NoError(t, err)

	timestamp5, err = time.Parse(time.RFC3339,
		"2001-12-21T22:08:41+00:00")
	require.NoError(t, err)

	res := []time.Time{
		timestamp0, timestamp1, timestamp2, timestamp3, timestamp4, timestamp5,
	}
	curTime, err := time.Parse(time.RFC3339, "2023-05-19T22:08:41+00:00")
	require.NoError(t, err)
	for i := 0; i < numRandomTimestamps; i++ {
		curTime = curTime.Add(1 * time.Second)
		for f := rand.Float32(); f < 0.5; f = rng.Float32() {
			curTime = curTime.Add(-900 * time.Millisecond)
		}
		res = append(res, curTime)
	}

	return res
}

// Mock upsert containing the key, old value and new value.
type mockUpsert struct {
	key            string
	curVal, newVal []byte
}
