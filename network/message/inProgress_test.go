package message

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/client/storage/versioned"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/comms/connect"
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
	kv := versioned.NewKV(make(ekv.Memstore))
	p := NewHandler(params.Network{Messages: params.Messages{
		MessageReceptionBuffLen:        20,
		MessageReceptionWorkerPoolSize: 20,
		MaxChecksInProcessMessage:      20,
		InProcessMessageWait:           time.Hour,
	}}, kv, nil).(*handler)

	msg := makeTestFormatMessages(1)[0]
	cid := id.NewIdFromString("clientID", id.User, t)
	fp := format.NewFingerprint([]byte("test"))
	mp := NewMockMsgProcessor(t)
	err := p.AddFingerprint(cid, fp, mp)
	if err != nil {
		t.Errorf("Failed to add fingerprint: %+v", err)
	}
	p.inProcess.Add(msg,
		&pb.RoundInfo{ID: 1, Timestamps: []uint64{0, 1, 2, 3}},
		interfaces.EphemeralIdentity{Source: cid})

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
