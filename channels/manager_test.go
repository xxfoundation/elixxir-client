////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/csprng"
	"os"
	"testing"
	"time"

	jww "github.com/spf13/jwalterweatherman"
)

func TestMain(m *testing.M) {
	// Many tests trigger WARN prints;, set the out threshold so the WARN prints
	// can be seen in the logs
	jww.SetStdoutThreshold(jww.LevelWarn)
	os.Exit(m.Run())
}

func TestManager_JoinChannel(t *testing.T) {
	mem := &mockEventModel{}

	m := NewManager(versioned.NewKV(ekv.MakeMemstore()),
		new(mockBroadcastClient),
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG),
		new(mockNameService), mem).(*manager)

	ch, _, err := newTestChannel("name", "description", m.rng.GetStream())
	if err != nil {
		t.Errorf("Failed to create new channel: %+v", err)
	}

	err = m.JoinChannel(ch)
	if err != nil {
		t.Fatalf("Join Channel Errored: %s", err)
	}

	if _, exists := m.channels[*ch.ReceptionID]; !exists {
		t.Errorf("Channel %s not added to channel map.", ch.Name)
	}

	//wait because the event model is called in another thread
	time.Sleep(1 * time.Second)

	if mem.joinedCh == nil {
		t.Errorf("the channel join call was not propogated to the event " +
			"model")
	}
}

func TestManager_LeaveChannel(t *testing.T) {
	mem := &mockEventModel{}

	m := NewManager(versioned.NewKV(ekv.MakeMemstore()),
		new(mockBroadcastClient),
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG),
		new(mockNameService), mem).(*manager)

	ch, _, err := newTestChannel("name", "description", m.rng.GetStream())
	if err != nil {
		t.Errorf("Failed to create new channel: %+v", err)
	}

	err = m.JoinChannel(ch)
	if err != nil {
		t.Fatalf("Join Channel Errored: %s", err)
	}

	err = m.LeaveChannel(ch.ReceptionID)
	if err != nil {
		t.Fatalf("Leave Channel Errored: %s", err)
	}

	if _, exists := m.channels[*ch.ReceptionID]; exists {
		t.Errorf("Channel %s still in map.", ch.Name)
	}

	//wait because the event model is called in another thread
	time.Sleep(1 * time.Second)

	if mem.leftCh == nil {
		t.Errorf("the channel join call was not propogated to the event " +
			"model")
	}
}
