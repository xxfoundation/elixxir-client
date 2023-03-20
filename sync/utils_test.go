package sync

import (
	"encoding/base64"
	"github.com/stretchr/testify/require"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/utils"
	"math/rand"
	"os"
	"strconv"
	"testing"
	"time"
)

const (
	baseDir  = "testDir/"
	password = "password"
)

func TestMain(m *testing.M) {
	utils.MakeDirs(baseDir, 0777)
	defer os.RemoveAll(baseDir)
	os.Exit(m.Run())
}

// dvcOffsetSerialEnc is the hardcoded deviceOffset.
const dvcOffsetSerialEnc = "WFhES1RYTE9HRFZDT0ZGU1RleUl3SWpvd0xDSXhJam94TENJeE1DSTZNVEFzSWpFeElqb3hNU3dpTVRJaU9qRXlMQ0l4TXlJNk1UTXNJakUwSWpveE5Dd2lNVFVpT2pFMUxDSXhOaUk2TVRZc0lqRTNJam94Tnl3aU1UZ2lPakU0TENJeE9TSTZNVGtzSWpJaU9qSXNJakl3SWpveU1Dd2lNakVpT2pJeExDSXlNaUk2TWpJc0lqSXpJam95TXl3aU1qUWlPakkwTENJeU5TSTZNalVzSWpJMklqb3lOaXdpTWpjaU9qSTNMQ0l5T0NJNk1qZ3NJakk1SWpveU9Td2lNeUk2TXl3aU16QWlPak13TENJek1TSTZNekVzSWpNeUlqb3pNaXdpTXpNaU9qTXpMQ0l6TkNJNk16UXNJak0xSWpvek5Td2lNellpT2pNMkxDSXpOeUk2TXpjc0lqTTRJam96T0N3aU16a2lPak01TENJMElqbzBMQ0kwTUNJNk5EQXNJalF4SWpvME1Td2lORElpT2pReUxDSTBNeUk2TkRNc0lqUTBJam8wTkN3aU5EVWlPalExTENJME5pSTZORFlzSWpRM0lqbzBOeXdpTkRnaU9qUTRMQ0kwT1NJNk5Ea3NJalVpT2pVc0lqVXdJam8xTUN3aU5URWlPalV4TENJMU1pSTZOVElzSWpVeklqbzFNeXdpTlRRaU9qVTBMQ0kxTlNJNk5UVXNJalUySWpvMU5pd2lOVGNpT2pVM0xDSTFPQ0k2TlRnc0lqVTVJam8xT1N3aU5pSTZOaXdpTmpBaU9qWXdMQ0kyTVNJNk5qRXNJall5SWpvMk1pd2lOak1pT2pZekxDSTJOQ0k2TmpRc0lqWTFJam8yTlN3aU5qWWlPalkyTENJMk55STZOamNzSWpZNElqbzJPQ3dpTmpraU9qWTVMQ0kzSWpvM0xDSTNNQ0k2TnpBc0lqY3hJam8zTVN3aU56SWlPamN5TENJM015STZOek1zSWpjMElqbzNOQ3dpTnpVaU9qYzFMQ0kzTmlJNk56WXNJamMzSWpvM055d2lOemdpT2pjNExDSTNPU0k2Tnprc0lqZ2lPamdzSWpnd0lqbzRNQ3dpT0RFaU9qZ3hMQ0k0TWlJNk9ESXNJamd6SWpvNE15d2lPRFFpT2pnMExDSTROU0k2T0RVc0lqZzJJam80Tml3aU9EY2lPamczTENJNE9DSTZPRGdzSWpnNUlqbzRPU3dpT1NJNk9Td2lPVEFpT2prd0xDSTVNU0k2T1RFc0lqa3lJam81TWl3aU9UTWlPamt6TENJNU5DSTZPVFFzSWprMUlqbzVOU3dpT1RZaU9qazJMQ0k1TnlJNk9UY3NJams0SWpvNU9Dd2lPVGtpT2prNWZRPT0="

// Full test for deviceOffset.
func TestDeviceOffset(t *testing.T) {
	dvcOffset := newDeviceOffset()
	require.Len(t, dvcOffset, 0)

	// Populate offset structure with data
	const numTests = 100
	for i := 0; i < numTests; i++ {
		dvcId := DeviceId(strconv.Itoa(i))
		dvcOffset[dvcId] = i
	}

	require.Len(t, dvcOffset, numTests)

	// Serialize device offset
	dvcOffsetSerial, err := dvcOffset.serialize()
	require.NoError(t, err)

	// Check that device offset matches hardcoded value
	require.Equal(t, dvcOffsetSerialEnc, base64.StdEncoding.EncodeToString(dvcOffsetSerial))

	// Deserialize offset
	deserial, err := deserializeDeviceOffset(dvcOffsetSerial)
	require.NoError(t, err)

	// Ensure deserialized offset matches original
	require.Equal(t, dvcOffset, deserial)
}

// makeTransactionLog is a utility function which generates a TransactionLog for
// testing purposes.
func makeTransactionLog(baseDir, password string, t *testing.T) *TransactionLog {
	// Construct local store
	fs, err := ekv.NewFilestore(baseDir, password)
	require.NoError(t, err)

	localStore, err := NewOrLoadEkvLocalStore(versioned.NewKV(fs))
	require.NoError(t, err)
	// Construct remote store
	remoteStore := NewFileSystemRemoteStorage(baseDir)

	// Construct device secret
	deviceSecret := []byte("deviceSecret")

	// Construct transaction log
	txLog, err := NewOrLoadTransactionLog(baseDir+"test.txt", localStore,
		remoteStore, deviceSecret, &CountingReader{count: 0})
	require.NoError(t, err)

	return txLog
}

// CountingReader is a platform-independent deterministic RNG that adheres to
// io.Reader.
type CountingReader struct {
	count uint8
}

// Read just counts until 254 then starts over again
func (c *CountingReader) Read(b []byte) (int, error) {
	for i := 0; i < len(b); i++ {
		c.count = (c.count + 1) % 255
		b[i] = c.count
	}
	return len(b), nil
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

	res := make([]time.Time, 0)

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

	res = []time.Time{
		timestamp0, timestamp1, timestamp2, timestamp3, timestamp4, timestamp5,
	}
	curTime := time.Now()
	for i := 0; i < numRandomTimestamps; i++ {
		curTime = curTime.Add(1 * time.Second)
		for f := rand.Float32(); f < 0.5; f = rand.Float32() {
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
