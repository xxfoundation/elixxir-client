package ud

import (
	"crypto/ed25519"
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

func TestChannelIDTracking(t *testing.T) {
	rngGen := fastRNG.NewStreamGenerator(1000, 10, csprng.NewSystemRNG)

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
	// doesn't work:
	//username, err := m.store.GetUsername()
	//require.NoError(t, err)

	udPubKeyBytes := m.user.GetCmix().GetInstance().
		GetPartialNdf().Get().UDB.DhPubKey

	myTestClientIDTracker := newclientIDTracker(comms, host, username,
		kv, m.user.GetReceptionIdentity(), ed25519.PublicKey(udPubKeyBytes), rngGen)

	//stopper, err := myTestClientIDTracker.Start()
	//require.NoError(t, err)

	err = myTestClientIDTracker.register()
	require.NoError(t, err)

	require.Equal(t, myTestClientIDTracker.GetUsername(), username)

	signature, lease := myTestClientIDTracker.GetChannelValidationSignature()
	t.Logf("signature %x lease %v", signature, lease)

	chanPubKey := myTestClientIDTracker.GetChannelPubkey()
	t.Logf("channel public key: %x", chanPubKey)

	message := []byte("hello world")
	signature2, err := myTestClientIDTracker.SignChannelMessage(message)
	require.NoError(t, err)

	t.Logf("signature2: %x", signature2)

	//_ = myTestClientIDTracker.ValidateChannelMessage(username, lease, pubKey, authorIDSignature)

	//err = stopper.Close()
	//require.NoError(t, err)
}
