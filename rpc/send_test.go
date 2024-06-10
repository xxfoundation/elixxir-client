////////////////////////////////////////////////////////////////////////////////
// Copyright © 2024 xx foundation                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found         //
// in the LICENSE file                                                        //
////////////////////////////////////////////////////////////////////////////////

package rpc

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"

	"github.com/stretchr/testify/require"
	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/cmix/message"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/nike/ecdh"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
)

func TestSend(t *testing.T) {
	rng := csprng.NewSystemRNG()
	serverID, err := generateRandomID(rng)
	require.NoError(t, err)

	serverPriv, serverPub := ecdh.ECDHNIKE.NewKeypair(rng)

	msgs := make(chan format.Message, 10)
	net := MockCmix(t, msgs)

	expMsg := []byte("Hello, World!")

	r := Send(net, serverID, serverPub, expMsg,
		cmix.GetDefaultCMIXParams())
	require.NotNil(t, net.processor)

	success := func(data []byte) {
		var prettyJSON bytes.Buffer
		err := json.Indent(&prettyJSON, data, "", "\t")
		if err != nil {
			t.Errorf("%+v", err)
		}
		t.Errorf("%s", prettyJSON.String())
	}
	fail := func(err error) {
		t.Errorf("%+v", err)
	}

	r.Callback(success, fail)

	cPub := ecdh.ECDHNIKE.NewEmptyPublicKey()
	kSz := ecdh.ECDHNIKE.PublicKeySize()
	var server *noise
	for m := range msgs {
		cm := reconstructCiphertext(m)
		var mid msgId
		copy(mid[:], cm[:msgIdSz])
		ct := cm[msgIdSz:]
		if server == nil {
			err := cPub.FromBytes(cm[:kSz])
			require.NoError(t, err)
			ct = cm[kSz:]
			server, err = startNoiseServer(serverPriv, cPub)
			require.NoError(t, err)
		}

		pt, err := server.ReadMessage(ct)
		require.NoError(t, err)
		rawpt, err := reconstructPartitions([][]byte{pt})
		require.NoError(t, err)
		rID, err := id.Unmarshal(rawpt[:id.ArrIDLen])
		require.NoError(t, err)
		msg := rawpt[id.ArrIDLen:]
		require.Equal(t, expMsg, msg)
		t.Logf("Message: %v, %s, %s", mid, rID, msg)
	}

	require.Equal(t, true, false)

}

func MockCmix(t *testing.T, msgs chan format.Message) *mockCmixServer {
	return &mockCmixServer{
		processor: nil,
		msgs:      msgs,
	}
}

type mockCmixServer struct {
	processor message.Processor
	msgs      chan format.Message
}

func (c *mockCmixServer) AddIdentityWithHistory(id *id.ID, validUntil,
	beginning time.Time, persistent bool,
	fallthroughProcessor message.Processor) {
	c.processor = fallthroughProcessor
}

func (c *mockCmixServer) AddIdentity(id *id.ID, validUntil time.Time,
	persistent bool, fallthroughProcessor message.Processor) {
	c.processor = fallthroughProcessor
}

func (c *mockCmixServer) RemoveIdentity(id *id.ID) {
	c.processor = nil
}

func (c *mockCmixServer) GetRoundResults(timeout time.Duration,
	roundCallback cmix.RoundEventCallback, roundList ...id.Round) {
	go func() {
		time.Sleep(time.Second * 3)
		result := make(map[id.Round]cmix.RoundResult)
		result[roundList[0]] = cmix.RoundResult{
			Status: cmix.Succeeded,
			Round: rounds.Round{
				ID: roundList[0],
			},
		}
		roundCallback(true, false, result)
	}()
}

func (c *mockCmixServer) RNGStreamGenerator() *fastRNG.StreamGenerator {
	s := fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG)
	return s
}

func (c *mockCmixServer) GetMaxMessageLength() int {
	// 4096 is the size of the prime we use
	emptyMsg := format.NewMessage(4096)
	return emptyMsg.ContentsSize()
}

func (c *mockCmixServer) SendManyWithAssembler(recipients []*id.ID,
	assembler cmix.ManyMessageAssembler, params cmix.CMIXParams) (
	rounds.Round, []ephemeral.Id, error) {

	rng := c.RNGStreamGenerator().GetStream()
	defer rng.Close()

	rnd := id.Round(8675309)
	msgs, err := assembler(rnd)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}

	ephIds := make([]ephemeral.Id, len(msgs))
	for i := 0; i < len(ephIds); i++ {
		rng.Read(ephIds[i][:])
	}

	go func() {
		for i := 0; i < len(msgs); i++ {
			fmsg := format.NewMessage(4096)
			fmsg.SetEphemeralRID(ephIds[i][:])
			fmsg.SetKeyFP(msgs[i].Fingerprint)
			fmsg.SetMac(msgs[i].Mac)
			fmsg.SetContents(msgs[i].Payload)
			c.msgs <- fmsg
		}
	}()

	r := rounds.Round{
		ID: rnd,
		Raw: &pb.RoundInfo{
			ID: uint64(rnd),
		},
	}

	return r, ephIds, nil
}
