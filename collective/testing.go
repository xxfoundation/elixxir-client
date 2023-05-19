package collective

import (
	"strings"
	"sync"
	"testing"
	"time"

	jww "github.com/spf13/jwalterweatherman"
	"github.com/stretchr/testify/require"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/csprng"
)

// params used by the testing KV
const TestingKVPath = "versionedKV_TestWorkDir"

// StandardPrefexs will be passed into tests for syncPrefixes
var StandardPrefexs = []string{StandardRemoteSyncPrefix}

// TestingKV returns a versioned ekv which can be used for testing. it does not do
// remote writes but maintains the proper interface
func TestingKV(t interface{}, kv ekv.KeyValue, syncPrefixes []string,
	remoteStore RemoteStore) versioned.KV {
	switch t.(type) {
	case *testing.T, *testing.M, *testing.B, *testing.PB:
		break
	default:
		jww.FATAL.Panicf("TestingKV is restricted to "+
			"testing only. Got %T", t)
	}
	rkv, _ := testingKV(t, kv, syncPrefixes, remoteStore)
	return rkv
}

func testingKV(t interface{}, kv ekv.KeyValue,
	syncPrefixes []string, remoteStore RemoteStore) (*versionedKV, *remoteWriter) {
	if t == nil {
		jww.FATAL.Printf("Cannot use testing KV in production")
	}
	txLog := makeTransactionLog(kv, TestingKVPath, remoteStore, t)
	return newVersionedKV(txLog, kv, syncPrefixes), txLog
}

// makeTransactionLog is a utility function which generates a remoteWriter for
// testing purposes.
func makeTransactionLog(kv ekv.KeyValue, baseDir string,
	remoteStore RemoteStore, x interface{}) *remoteWriter {
	switch x.(type) {
	case *testing.T, *testing.M, *testing.B, *testing.PB:
		break
	default:
		jww.FATAL.Panicf("makeTransactionLof is restricted to testing "+
			"only. Got %T", x)
	}

	t := x.(testing.TB)

	// Construct device secret
	deviceSecret := []byte("deviceSecret")

	rngGen := fastRNG.NewStreamGenerator(1, 1, NewCountingReader)

	rng := rngGen.GetStream()
	defer rng.Close()
	deviceID, err := InitInstanceID(kv, rng)
	require.NoError(t, err)

	crypt := &deviceCrypto{
		secret: deviceSecret,
		rngGen: rngGen,
	}

	// Construct mutate log
	txLog, err := newRemoteWriter(baseDir, deviceID,
		remoteStore, crypt, kv)
	if err != nil {
		jww.FATAL.Panicf("Cannot continue when creation of remote writer "+
			"fails: %+v", err)
	}

	return txLog
}

type mockRemote struct {
	lck       sync.Mutex
	data      map[string][]byte
	lastMod   map[string]time.Time
	lastWrite time.Time
}

func newMockRemote() *mockRemote {
	return &mockRemote{
		data:      make(map[string][]byte),
		lastMod:   make(map[string]time.Time),
		lastWrite: time.Unix(0, 0),
	}
}

func (m *mockRemote) Read(path string) ([]byte, error) {
	m.lck.Lock()
	defer m.lck.Unlock()
	jww.ERROR.Printf("Read: %s", path)
	return m.data[path], nil
}

func (m *mockRemote) Write(path string, data []byte) error {
	m.lck.Lock()
	defer m.lck.Unlock()
	jww.ERROR.Printf("Write: %s", path)

	m.lastWrite = time.Now()
	m.data[path] = data
	m.lastMod[path] = time.Now()
	return nil
}

func (m *mockRemote) ReadDir(path string) ([]string, error) {
	m.lck.Lock()
	defer m.lck.Unlock()
	jww.ERROR.Printf("ReadDir: %s", path)

	paths := make([]string, 0)
	for k := range m.data {
		if strings.HasPrefix(k, path) {
			paths = append(paths, k)
		}
	}
	return paths, nil
}

func (m *mockRemote) GetLastModified(path string) (time.Time, error) {
	m.lck.Lock()
	defer m.lck.Unlock()
	jww.ERROR.Printf("GetLastModified: %s", path)
	lastMod, ok := m.lastMod[path]
	if ok {
		return lastMod, nil
	}
	return time.Unix(0, 0), nil
}

func (m *mockRemote) GetLastWrite() (time.Time, error) {
	return m.lastWrite, nil
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
