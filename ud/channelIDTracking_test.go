package ud

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"

	"gitlab.com/elixxir/client/event"
	"gitlab.com/elixxir/client/storage/versioned"
	store "gitlab.com/elixxir/client/ud/store"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/ekv"
)

func TestLoadSaveRegistration(t *testing.T) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	lease := time.Now()
	signature := make([]byte, 64)
	reg := newRegistrationDisk(publicKey, privateKey, lease, signature)

	kv := versioned.NewKV(ekv.MakeMemstore())

	registrationDisk, err := loadRegistrationDisk(kv)
	require.Error(t, err)
	t.Logf("err is %v", err)

	// kv api sucks... forcing me to do this:
	objectNotFoundErr := "object not found"
	require.Equal(t, err.Error(), objectNotFoundErr)

	err = saveRegistrationDisk(kv, reg)
	require.NoError(t, err)

	registrationDisk, err = loadRegistrationDisk(kv)
	require.NoError(t, err)
	require.Equal(t, registrationDisk, reg)
}

func TestChannelIDTracking(t *testing.T) {
	rngGen := fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG)

	t.Logf("rngGen: %v", rngGen)

	// AddHost
	stream := rngGen.GetStream()
	privKey, err := rsa.GenerateKey(stream, 1024)
	require.NoError(t, err)

	tnm := newTestNetworkManager(t)
	managerkv := versioned.NewKV(ekv.MakeMemstore())
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

	//

	kv := versioned.NewKV(ekv.MakeMemstore())
	comms := new(mockComms)
	username := "Alice"

	/*
		udPubKeyBytes := m.user.GetCmix().GetInstance().
			GetPartialNdf().Get().UDB.DhPubKey
	*/

	udPubKey, udPrivKey, err := ed25519.GenerateKey(stream)
	require.NoError(t, err)

	myTestClientIDTracker := newclientIDTracker(
		comms, host, username,
		kv, m.user.GetReceptionIdentity(),
		udPubKey, rngGen)

	rsaPrivKey, err := myTestClientIDTracker.receptionIdentity.GetRSAPrivateKey()
	require.NoError(t, err)

	comms.SetUserRSAPubKey(rsaPrivKey.GetPublic())
	comms.SetUDEd25519PrivateKey(&udPrivKey)
	comms.SetUserEd25519PubKey(myTestClientIDTracker.registrationDisk.GetPublicKey())
	comms.SetUsername(username)

	//sig, _ := myTestClientIDTracker.registrationDisk.GetLeaseSignature()
	// XXX bad signature
	sig := make([]byte, 64)
	stream.Read(sig)

	err = myTestClientIDTracker.register()
	require.NoError(t, err)
}
