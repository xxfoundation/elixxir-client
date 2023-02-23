package sync

import (
	"github.com/stretchr/testify/require"
	"gitlab.com/xx_network/primitives/utils"
	"math/rand"
	"os"
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

// makeTransactionLog is a utility function which generates a TransactionLog for
// testing purposes.
func makeTransactionLog(baseDir, password string, t *testing.T) *TransactionLog {
	// Construct local store
	localStore, err := NewEkvLocalStore(baseDir, password)
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
