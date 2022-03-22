package message

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/network/gateway"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
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

// Hear is called to exercise the listener, passing in the data as an item.
func (l TestListener) Hear(item message.Receive) {
	l.ch <- true
}

// Name returns a name; used for debugging.
func (l TestListener) Name() string {
	return "TEST LISTENER FOR GARBLED MESSAGES"
}

func Test_pickup_CheckInProgressMessages(t *testing.T) {
	sess := storage.InitTestingSession(t)
	rngGen := fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG)

	poolParams := gateway.DefaultPoolParams()
	poolParams.MaxPoolSize = 1
	sender, err := gateway.NewSender(
		poolParams, rngGen, getNDF(), &MockSendCMIXComms{t}, sess, nil)
	if err != nil {
		t.Errorf("Failed to make new sender: %+v", err)
	}
	newPickup := NewPickup(params.Network{Messages: params.Messages{
		MessageReceptionBuffLen:        20,
		MessageReceptionWorkerPoolSize: 20,
		MaxChecksInProcessMessage:      20,
		InProcessMessageWait:           time.Hour,
	}}, sender, sess, nil)
	p := newPickup.(*pickup)

	// rng := csprng.NewSystemRNG()
	// partnerSIDHPrivKey := util.NewSIDHPrivateKey(sidh.KeyVariantSidhA)
	// partnerSIDHPubKey := util.NewSIDHPublicKey(sidh.KeyVariantSidhA)
	// partnerSIDHPrivKey.Generate(rng)
	// partnerSIDHPrivKey.GeneratePublicKey(partnerSIDHPubKey)
	// mySIDHPrivKey := util.NewSIDHPrivateKey(sidh.KeyVariantSidhB)
	// mySIDHPubKey := util.NewSIDHPublicKey(sidh.KeyVariantSidhB)
	// mySIDHPrivKey.Generate(rng)
	// mySIDHPrivKey.GeneratePublicKey(mySIDHPubKey)
	//
	// e2ekv := i.Session.E2e()
	// err = e2ekv.AddPartner(sess2.GetUser().TransmissionID,
	// 	sess2.E2e().GetDHPublicKey(), e2ekv.GetDHPrivateKey(),
	// 	partnerSIDHPubKey, mySIDHPrivKey,
	// 	params.GetDefaultE2ESessionParams(),
	// 	params.GetDefaultE2ESessionParams())
	// if err != nil {
	// 	t.Errorf("Failed to add e2e partner: %+v", err)
	// 	t.FailNow()
	// }
	//
	// preimage := edge.Preimage{
	// 	Data:   []byte{0},
	// 	Type:   "test",
	// 	Source: nil,
	// }
	// p.Session.GetEdge().Add(preimage, sess2.GetUser().ReceptionID)
	//
	// err = sess2.E2e().AddPartner(sess.GetUser().TransmissionID,
	// 	sess.E2e().GetDHPublicKey(), sess2.E2e().GetDHPrivateKey(),
	// 	mySIDHPubKey, partnerSIDHPrivKey,
	// 	params.GetDefaultE2ESessionParams(),
	// 	params.GetDefaultE2ESessionParams())
	// if err != nil {
	// 	t.Errorf("Failed to add e2e partner: %+v", err)
	// 	t.FailNow()
	// }
	// partner1, err := sess2.E2e().GetPartner(sess.GetUser().ReceptionID)
	// if err != nil {
	// 	t.Errorf("Failed to get partner: %+v", err)
	// 	t.FailNow()
	// }

	// msg := format.NewMessage(format.MinimumPrimeSize)
	//
	// key, err := partner1.GetKeyForSending(params.Standard)
	// if err != nil {
	// 	t.Errorf("failed to get key: %+v", err)
	// 	t.FailNow()
	// }
	//
	// contents := make([]byte, msg.ContentsSize())
	// prng := rand.New(rand.NewSource(42))
	// prng.Read(contents)
	// contents[len(contents)-1] = 0
	// fmp := parse.FirstMessagePartFromBytes(contents)
	// binary.BigEndian.PutUint32(fmp.Type, message.Raw)
	// fmp.NumParts[0] = uint8(1)
	// binary.BigEndian.PutUint16(fmp.Len, 256)
	// fmp.Part[0] = 0
	// ts, err := netTime.Now().MarshalBinary()
	// if err != nil {
	// 	t.Errorf("failed to martial ts: %+v", err)
	// }
	// copy(fmp.Timestamp, ts)
	// msg.SetContents(fmp.Bytes())
	// encryptedMsg := key.Encrypt(msg)
	// msg.SetIdentityFP(fingerprint.IdentityFP(msg.GetContents(), preimage.Data))
	// i.Session.GetGarbledMessages().Add(encryptedMsg)

	msg := makeTestFormatMessages(1)[0]

	cid := id.NewIdFromString("clientID", id.User, t)
	fp := format.NewFingerprint([]byte("test"))
	mp := NewMockMsgProcessor(t)
	err = p.AddFingerprint(cid, fp, mp)
	if err != nil {
		t.Errorf("Failed to add fingerprint: %+v", err)
	}
	p.inProcess.Add(msg,
		&pb.RoundInfo{ID: 1, Timestamps: []uint64{0, 1, 2, 3}},
		interfaces.Identity{Source: cid})

	stop := stoppable.NewSingle("stop")
	go p.recheckInProgressRunner(stop)

	p.CheckInProgressMessages()

	select {
	case <-time.After(1000 * time.Millisecond):
		t.Error("Didn't hear anything")
	case <-p.messageReception:
		t.Log("Heard something")
	}

	err = stop.Close()
	if err != nil {
		t.Errorf("Failed to close stoppable: %+v", err)
	}
}
