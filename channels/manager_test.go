////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"fmt"
	"gitlab.com/elixxir/client/broadcast"
	"gitlab.com/elixxir/client/storage/versioned"
	broadcast2 "gitlab.com/elixxir/crypto/broadcast"
	cryptoChannel "gitlab.com/elixxir/crypto/channel"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"math/rand"
	"os"
	"sync"
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
	rng := rand.New(rand.NewSource(64))

	pi, err := cryptoChannel.GenerateIdentity(rng)
	if err != nil {
		t.Fatalf(err.Error())
	}

	mFace, err := NewManager(pi, versioned.NewKV(ekv.MakeMemstore()),
		new(mockBroadcastClient),
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG),
		mockEventModelBuilder)
	if err != nil {
		t.Errorf(err.Error())
	}

	m := mFace.(*manager)
	mem := m.events.model.(*mockEventModel)

	ch, _, err := newTestChannel(
		"name", "description", m.rng.GetStream(), broadcast2.Public)
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

	rng := rand.New(rand.NewSource(64))

	pi, err := cryptoChannel.GenerateIdentity(rng)
	if err != nil {
		t.Fatalf(err.Error())
	}

	mFace, err := NewManager(pi, versioned.NewKV(ekv.MakeMemstore()),
		new(mockBroadcastClient),
		fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG),
		mockEventModelBuilder)
	if err != nil {
		t.Errorf(err.Error())
	}

	m := mFace.(*manager)
	mem := m.events.model.(*mockEventModel)

	ch, _, err := newTestChannel(
		"name", "description", m.rng.GetStream(), broadcast2.Public)
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

func TestManager_GetChannels(t *testing.T) {
	m := &manager{
		channels: make(map[id.ID]*joinedChannel),
		mux:      sync.RWMutex{},
	}

	rng := fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG)

	numtests := 10

	chList := make(map[id.ID]interface{})

	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("testChannel_%d", numtests)
		s := rng.GetStream()
		tc, _, err := newTestChannel(name, "blarg", s, broadcast2.Public)
		s.Close()
		if err != nil {
			t.Fatalf("failed to generate channel %s", name)
		}
		bc, err := broadcast.NewBroadcastChannel(tc, new(mockBroadcastClient), rng)
		if err != nil {
			t.Fatalf("failed to generate broadcast %s", name)
		}
		m.channels[*tc.ReceptionID] = &joinedChannel{broadcast: bc}
		chList[*tc.ReceptionID] = nil
	}

	receivedChList := m.GetChannels()

	for _, receivedCh := range receivedChList {
		if _, exists := chList[*receivedCh]; !exists {
			t.Errorf("Channel was not returned")
		}
	}
}

func TestManager_GetChannel(t *testing.T) {
	m := &manager{
		channels: make(map[id.ID]*joinedChannel),
		mux:      sync.RWMutex{},
	}

	rng := fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG)

	numtests := 10

	chList := make([]*id.ID, 0, numtests)

	for i := 0; i < 10; i++ {
		name := fmt.Sprintf("testChannel_%d", numtests)
		s := rng.GetStream()
		tc, _, err := newTestChannel(name, "blarg", s, broadcast2.Public)
		s.Close()
		if err != nil {
			t.Fatalf("failed to generate channel %s", name)
		}
		bc, err := broadcast.NewBroadcastChannel(tc, new(mockBroadcastClient), rng)
		if err != nil {
			t.Fatalf("failed to generate broadcast %s", name)
		}
		m.channels[*tc.ReceptionID] = &joinedChannel{broadcast: bc}
		chList = append(chList, tc.ReceptionID)
	}

	for i, receivedCh := range chList {
		ch, err := m.GetChannel(receivedCh)
		if err != nil {
			t.Errorf("Channel %d failed to be gotten", i)
		} else if !ch.ReceptionID.Cmp(receivedCh) {
			t.Errorf("Channel %d Get returned wrong channel", i)
		}
	}
}

func TestManager_GetChannel_BadChannel(t *testing.T) {
	m := &manager{
		channels: make(map[id.ID]*joinedChannel),
		mux:      sync.RWMutex{},
	}

	numtests := 10

	chList := make([]*id.ID, 0, numtests)

	for i := 0; i < 10; i++ {
		chId := &id.ID{}
		chId[0] = byte(i)
		chList = append(chList, chId)
	}

	for i, receivedCh := range chList {
		_, err := m.GetChannel(receivedCh)
		if err == nil {
			t.Errorf("Channel %d returned when it doesnt exist", i)
		}
	}
}
