package collective

import (
	jww "github.com/spf13/jwalterweatherman"
	"github.com/stretchr/testify/require"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/netTime"
	"sync"
	"testing"
	"time"
)

// params used by the testing KV
const TestingKVPath = "versionedKV_TestWorkDir"

// StandardPrefexs will be passed into tests for syncPrefixes
var StandardPrefexs = []string{StandardRemoteSyncPrefix}

// TestingKV returns a versioned ekv which can be used for testing. it does not do
// remote writes but maintains the proper interface
func TestingKV(t *testing.T, kv ekv.KeyValue, syncPrefixes []string) versioned.KV {
	rkv, _ := testingKV(t, kv, syncPrefixes)
	return rkv
}

func testingKV(t *testing.T, kv ekv.KeyValue,
	syncPrefixes []string) (*versionedKV, *remoteWriter) {
	if t == nil {
		jww.FATAL.Printf("Cannot use testing KV in production")
	}
	txLog := makeTransactionLog(kv, TestingKVPath, t)
	return newVersionedKV(txLog, kv, syncPrefixes), txLog
}

// makeTransactionLog is a utility function which generates a remoteWriter for
// testing purposes.
func makeTransactionLog(kv ekv.KeyValue, baseDir string, t *testing.T) *remoteWriter {

	// Construct remote store
	remoteStore := &mockRemote{data: make(map[string][]byte)}

	// Construct device secret
	deviceSecret := []byte("deviceSecret")

	rngGen := fastRNG.NewStreamGenerator(1, 1, NewCountingReader)

	rng := rngGen.GetStream()
	defer rng.Close()
	deviceID, err := InitInstanceID(kv, rng)

	crypt := &deviceCrypto{
		secret: deviceSecret,
		rngGen: rngGen,
	}

	// Construct mutate log
	txLog, err := newRemoteWriter(baseDir+"test.txt", deviceID,
		remoteStore, crypt, kv)
	require.NoError(t, err)

	return txLog
}

type mockRemote struct {
	lck  sync.Mutex
	data map[string][]byte
}

func (m *mockRemote) Read(path string) ([]byte, error) {
	m.lck.Lock()
	defer m.lck.Unlock()
	return m.data[path], nil
}

func (m *mockRemote) Write(path string, data []byte) error {
	m.lck.Lock()
	defer m.lck.Unlock()
	m.data[path] = append(m.data[path], data...)
	return nil
}

func (m *mockRemote) ReadDir(path string) ([]string, error) {
	panic("unimplemented")
}

func (m *mockRemote) GetLastModified(path string) (time.Time, error) {
	return netTime.Now(), nil
}

func (m *mockRemote) GetLastWrite() (time.Time, error) {
	return netTime.Now(), nil
}

// CountingReader is a platform-independent deterministic RNG that adheres to
// io.Reader.
type CountingReader struct {
	count uint8
}

func NewCountingReader() csprng.Source {
	return &CountingReader{count: 0}
}

// Read just counts until 254 then starts over again
func (c *CountingReader) Read(b []byte) (int, error) {
	for i := 0; i < len(b); i++ {
		c.count = (c.count + 1) % 255
		b[i] = c.count
	}
	return len(b), nil
}

func (c *CountingReader) SetSeed(s []byte) error {
	return nil
}
