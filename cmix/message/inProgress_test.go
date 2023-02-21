////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package message

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v4/stoppable"
	"gitlab.com/elixxir/client/v4/storage/versioned"
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

func TestHandler_CheckInProgressMessages(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	h := NewHandler(Params{
		MessageReceptionBuffLen:        20,
		MessageReceptionWorkerPoolSize: 20,
		MaxChecksInProcessMessage:      20,
		InProcessMessageWait:           time.Hour,
	}, kv, nil, nil).(*handler)

	msg := makeTestFormatMessages(1)[0]
	cid := id.NewIdFromString("clientID", id.User, t)
	fp := format.NewFingerprint([]byte("test"))
	mp := NewMockMsgProcessor(t)
	err := h.AddFingerprint(cid, fp, mp)
	if err != nil {
		t.Errorf("Failed to add fingerprint: %+v", err)
	}
	h.inProcess.Add(msg,
		&pb.RoundInfo{ID: 1, Timestamps: []uint64{0, 1, 2, 3},
			Topology: [][]byte{{1}, {2}}},
		receptionID.EphemeralIdentity{Source: cid})

	stop := stoppable.NewSingle("stop")
	go h.recheckInProgressRunner(stop)

	h.CheckInProgressMessages()

	select {
	case <-time.After(1 * time.Second):
		t.Error("Didn't hear anything")
	case <-h.messageReception:
		t.Log("Heard something")
	}

	err = stop.Close()
	if err != nil {
		t.Errorf("Failed to close stoppable: %+v", err)
	}
}
