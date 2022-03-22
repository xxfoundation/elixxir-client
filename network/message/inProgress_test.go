package message

import (
	"encoding/binary"
	"github.com/cloudflare/circl/dh/sidh"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/network/gateway"
	"gitlab.com/elixxir/client/network/internal"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/storage/edge"
	util "gitlab.com/elixxir/client/storage/utility"
	"gitlab.com/elixxir/client/switchboard"
	"gitlab.com/elixxir/comms/client"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/fingerprint"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/netTime"
	"math/rand"
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	jww.SetStdoutThreshold(jww.LevelTrace)
	connect.TestingOnlyDisableTLS = true
	os.Exit(m.Run())
}

type TestListener struct {
	ch chan bool
}

// the Hear function is called to exercise the listener, passing in the
// data as an item
func (l TestListener) Hear(item message.Receive) {
	l.ch <- true
}

// Returns a name, used for debugging
func (l TestListener) Name() string {
	return "TEST LISTENER FOR GARBLED MESSAGES"
}

func TestManager_CheckGarbledMessages(t *testing.T) {
	sess1 := storage.InitTestingSession(t)

	sess2 := storage.InitTestingSession(t)

	sw := switchboard.New()
	l := TestListener{
		ch: make(chan bool),
	}
	sw.RegisterListener(sess2.GetUser().TransmissionID, message.Raw, l)
	comms, err := client.NewClientComms(sess1.GetUser().TransmissionID, nil, nil, nil)
	if err != nil {
		t.Errorf("Failed to start client comms: %+v", err)
	}
	i := internal.Internal{
		Session:          sess1,
		Switchboard:      sw,
		Rng:              fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG),
		Comms:            comms,
		Health:           nil,
		TransmissionID:   sess1.GetUser().TransmissionID,
		Instance:         nil,
		NodeRegistration: nil,
	}
	p := gateway.DefaultPoolParams()
	p.MaxPoolSize = 1
	sender, err := gateway.NewSender(p, i.Rng, getNDF(), &MockSendCMIXComms{t: t}, i.Session, nil)
	if err != nil {
		t.Errorf(err.Error())
	}
	m := NewManager(i, params.Network{Messages: params.Messages{
		MessageReceptionBuffLen:        20,
		MessageReceptionWorkerPoolSize: 20,
		MaxChecksInProcessMessage:      20,
		InProcessMessageWait:           time.Hour,
	}}, nil, sender)

	rng := csprng.NewSystemRNG()
	partnerSIDHPrivKey := util.NewSIDHPrivateKey(sidh.KeyVariantSidhA)
	partnerSIDHPubKey := util.NewSIDHPublicKey(sidh.KeyVariantSidhA)
	partnerSIDHPrivKey.Generate(rng)
	partnerSIDHPrivKey.GeneratePublicKey(partnerSIDHPubKey)
	mySIDHPrivKey := util.NewSIDHPrivateKey(sidh.KeyVariantSidhB)
	mySIDHPubKey := util.NewSIDHPublicKey(sidh.KeyVariantSidhB)
	mySIDHPrivKey.Generate(rng)
	mySIDHPrivKey.GeneratePublicKey(mySIDHPubKey)

	e2ekv := i.Session.E2e()
	err = e2ekv.AddPartner(sess2.GetUser().TransmissionID,
		sess2.E2e().GetDHPublicKey(), e2ekv.GetDHPrivateKey(),
		partnerSIDHPubKey, mySIDHPrivKey,
		params.GetDefaultE2ESessionParams(),
		params.GetDefaultE2ESessionParams())
	if err != nil {
		t.Errorf("Failed to add e2e partner: %+v", err)
		t.FailNow()
	}

	preimage := edge.Preimage{
		Data:   []byte{0},
		Type:   "test",
		Source: nil,
	}
	m.Session.GetEdge().Add(preimage, sess2.GetUser().ReceptionID)

	err = sess2.E2e().AddPartner(sess1.GetUser().TransmissionID,
		sess1.E2e().GetDHPublicKey(), sess2.E2e().GetDHPrivateKey(),
		mySIDHPubKey, partnerSIDHPrivKey,
		params.GetDefaultE2ESessionParams(),
		params.GetDefaultE2ESessionParams())
	if err != nil {
		t.Errorf("Failed to add e2e partner: %+v", err)
		t.FailNow()
	}
	partner1, err := sess2.E2e().GetPartner(sess1.GetUser().ReceptionID)
	if err != nil {
		t.Errorf("Failed to get partner: %+v", err)
		t.FailNow()
	}

	msg := format.NewMessage(m.Session.Cmix().GetGroup().GetP().ByteLen())

	key, err := partner1.GetKeyForSending(params.Standard)
	if err != nil {
		t.Errorf("failed to get key: %+v", err)
		t.FailNow()
	}

	contents := make([]byte, msg.ContentsSize())
	prng := rand.New(rand.NewSource(42))
	prng.Read(contents)
	contents[len(contents)-1] = 0
	fmp := parse.FirstMessagePartFromBytes(contents)
	binary.BigEndian.PutUint32(fmp.Type, uint32(message.Raw))
	fmp.NumParts[0] = uint8(1)
	binary.BigEndian.PutUint16(fmp.Len, 256)
	fmp.Part[0] = 0
	ts, err := netTime.Now().MarshalBinary()
	if err != nil {
		t.Errorf("failed to martial ts: %+v", err)
	}
	copy(fmp.Timestamp, ts)
	msg.SetContents(fmp.Bytes())
	encryptedMsg := key.Encrypt(msg)
	msg.SetIdentityFP(fingerprint.IdentityFP(msg.GetContents(), preimage.Data))
	i.Session.GetGarbledMessages().Add(encryptedMsg)

	stop := stoppable.NewSingle("stop")
	go m.processGarbledMessages(stop)

	m.CheckGarbledMessages()

	ticker := time.NewTicker(time.Second)
	select {
	case <-ticker.C:
		t.Error("Didn't hear anything")
	case <-l.ch:
		t.Log("Heard something")
	}

}
