package ud

import (
	"crypto/ed25519"
	"crypto/rand"
	"gitlab.com/elixxir/client/v4/storage/utility"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"gitlab.com/elixxir/crypto/rsa"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"

	"gitlab.com/elixxir/client/v4/event"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	store "gitlab.com/elixxir/client/v4/ud/store"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/ekv"
)

func TestSignChannelMessage(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	reg := registrationDisk{
		PublicKey:  publicKey,
		PrivateKey: privateKey,
		Lease:      0,
	}
	c := &clientIDTracker{
		registrationDisk: &reg,
	}

	message := []byte("hello world")
	sig, err := c.SignChannelMessage(message)
	require.NoError(t, err)

	require.True(t, ed25519.Verify(publicKey, message, sig))
}

func TestNewRegistrationDisk(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	lease := time.Now().UnixNano()

	signature := make([]byte, 64)
	reg := newRegistrationDisk(publicKey, privateKey, time.Unix(0, lease), signature)
	require.Equal(t, reg.PublicKey, publicKey)
	require.Equal(t, reg.PrivateKey, privateKey)
	require.Equal(t, reg.Signature, signature)
	require.Equal(t, reg.Lease, lease)
}

func TestLoadSaveRegistration(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	lease := time.Now()
	signature := make([]byte, 64)
	reg := newRegistrationDisk(publicKey, privateKey, lease, signature)

	kv := &utility.KV{Local: versioned.NewKV(ekv.MakeMemstore())}

	registrationDisk, err := loadRegistrationDisk(kv)
	require.Error(t, err)
	require.False(t, kv.Exists(err))

	err = saveRegistrationDisk(kv, reg)
	require.NoError(t, err)

	registrationDisk, err = loadRegistrationDisk(kv)
	require.NoError(t, err)
	require.Equal(t, registrationDisk, reg)
}

func TestChannelIDTracking(t *testing.T) {
	rngGen := fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG)

	// comms AddHost
	stream := rngGen.GetStream()
	sch := rsa.GetScheme()
	privKey, err := sch.Generate(stream, 1024)
	require.NoError(t, err)

	tnm := newTestNetworkManager(t)
	managerkv := &utility.KV{Local: versioned.NewKV(ekv.MakeMemstore())}
	udStore, err := store.NewOrLoadStore(managerkv)
	m := &Manager{
		user: mockE2e{
			grp:     getGroup(),
			events:  event.NewEventManager(),
			rng:     rngGen,
			kv:      managerkv,
			network: tnm,
			t:       t,
			key:     privKey,
		},
		store: udStore,
		comms: &mockComms{},
	}

	netDef := m.getCmix().GetInstance().GetPartialNdf().Get()
	udID, err := id.Unmarshal(netDef.UDB.ID)
	require.NoError(t, err)

	params := connect.GetDefaultHostParams()
	params.AuthEnabled = false
	params.SendTimeout = 20 * time.Second

	host, err := m.comms.AddHost(udID, netDef.UDB.Address,
		[]byte(netDef.UDB.Cert), params)
	require.NoError(t, err)

	// register

	kv := &utility.KV{Local: versioned.NewKV(ekv.MakeMemstore())}
	comms := new(mockComms)
	username := "Alice"

	udPubKey, udPrivKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	rsaPrivKey, err := m.user.GetReceptionIdentity().GetRSAPrivateKey()
	require.NoError(t, err)

	comms.SetUserRSAPubKey(rsaPrivKey.Public())
	comms.SetUDEd25519PrivateKey(&udPrivKey)
	comms.SetUsername(username)

	myTestClientIDTracker := newclientIDTracker(
		comms, host, username,
		kv, m.user.GetReceptionIdentity(),
		udPubKey, rngGen)

	err = myTestClientIDTracker.register()
	require.NoError(t, err)
}
