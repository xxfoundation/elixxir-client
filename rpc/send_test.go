////////////////////////////////////////////////////////////////////////////////
// Copyright © 2024 xx foundation                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found         //
// in the LICENSE file                                                        //
////////////////////////////////////////////////////////////////////////////////

package rpc

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"

	"github.com/stretchr/testify/require"
	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/cmix/identity"
	"gitlab.com/elixxir/client/v4/cmix/identity/receptionID"
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
	net := MockCmix(t)

	net.AddIdentityWithHistory(serverID, identity.Forever,
		beginningOfTime, true, &mockRPCServer{msgs: msgs})

	expMsg := []byte("Hello, World!")

	r := Send(net, serverID, serverPub, expMsg,
		cmix.GetDefaultCMIXParams())
	rObj := r.(*response)
	require.NotNil(t, net.processor)

	responses := make([][]byte, 0)

	success := func(data []byte) {
		var prettyJSON bytes.Buffer
		err := json.Indent(&prettyJSON, data, "", "\t")
		require.NoError(t, err)
		t.Logf("%s", prettyJSON.String())
		responses = append(responses, data)
	}
	fail := func(err error) {
		require.NoError(t, err)
	}

	r.Callback(success, fail)

	// Receive the query from the client, and decrypt the message
	cPub := ecdh.ECDHNIKE.NewEmptyPublicKey()
	kSz := ecdh.ECDHNIKE.PublicKeySize()
	var server *noise
	m := <-msgs
	cm := reconstructCiphertext(m)
	err = cPub.FromBytes(cm[:kSz])
	require.NoError(t, err)
	ct := cm[kSz:]
	server, err = startNoiseServer(serverPriv, cPub)
	require.NoError(t, err)
	pt, err := server.ReadMessage(ct)
	require.NoError(t, err)
	rawpt, err := reconstructPartitions([][]byte{pt})
	require.NoError(t, err)
	rID, err := id.Unmarshal(rawpt[:id.ArrIDLen])
	require.NoError(t, err)
	msg := rawpt[id.ArrIDLen:]
	require.Equal(t, expMsg, msg)
	t.Logf("Message: %s, %s", rID, msg)

	// Reply to the client, sending 11 messages of response back
	// in reverse order to test the message reconstruction.
	maxPayloadSz := uint64(maxPayloadLen(net))
	// NOTE: The server does not prepend the ephemeral pubkey
	// ephKeySz := uint64(len(serverPub.Bytes()))
	headerSz := maxPayloadSz - handshakeCiphertextOverhead - msgIdSz
	otherSz := maxPayloadSz - channelCiphertextOverhead - msgIdSz
	reply := make([]byte, otherSz*10)
	for i := 0; i < len(reply); i++ {
		reply[i] = msg[i%len(msg)]
	}

	parts := partitionMessage(reply, headerSz, otherSz)
	require.Equal(t, len(parts), 11)

	// Encrypt parts
	ciphertexts := make([][]byte, len(parts))
	for i := 0; i < len(parts); i++ {
		ct := server.WriteMessage(parts[i])
		// prefix the id of the previous message to this one
		if i == 0 {
			firstId := genResponseMsgID(rawpt, server.SharedKey)
			ct = append(firstId[:], ct...)
		} else {
			prevId := genResponseMsgID(parts[i-1], server.SharedKey)
			ct = append(prevId[:], ct...)
		}
		ciphertexts[i] = ct
	}

	for i := len(ciphertexts); i != 0; i-- {
		idx := i - 1
		payloadLen := maxPayloadLen(net)
		fpBytes, encryptedPayload, mac, err := createCMIXFields(
			ciphertexts[idx], payloadLen, rng)
		require.NoError(t, err)

		fp := format.NewFingerprint(fpBytes)

		fmsg := format.NewMessage(4096)
		fmsg.SetMac(mac)
		fmsg.SetKeyFP(fp)
		fmsg.SetContents(encryptedPayload)

		eID, _, _, _ := ephemeral.GetId(rID, 16, time.Now().UnixMilli())
		recID := receptionID.EphemeralIdentity{
			EphId:  eID,
			Source: rID,
		}
		net.processor[*rObj.myID].Process(fmsg, nil, nil, recID,
			rounds.Round{ID: 8675310})
	}

	d := r.Wait()
	require.Equal(t, reply, d)

	// Wait for our receiver func to record it properly
	time.Sleep(100 * time.Millisecond)

	rTypes := []string{"SentMessage", "RoundResults", "QueryResponse"}
	objs := make(map[string]interface{})
	for i := 0; i < len(responses); i++ {
		o := make(map[string]interface{})
		err := json.Unmarshal(responses[i], &o)
		require.NoError(t, err)
		tyName := o["type"].(string)
		objs[tyName] = o["response"]
	}

	// Check that the results include all the expected types
	for i := 0; i < len(rTypes); i++ {
		o, ok := objs[rTypes[i]]
		require.True(t, ok, fmt.Sprintf("missing %s: %v",
			rTypes[i], o))
		if rTypes[i] == "QueryResponse" {
			qr := o.(map[string]interface{})
			r, err := base64.RawStdEncoding.DecodeString(
				qr["message"].(string))
			require.NoError(t, err)
			require.Equal(t, reply, r)
		}
	}

}

func MockCmix(t *testing.T) *mockCmixServer {
	return &mockCmixServer{
		processor: make(map[id.ID]message.Processor),
		curRnd:    8675309,
	}
}

type mockCmixServer struct {
	processor map[id.ID]message.Processor
	curRnd    int
}

func (c *mockCmixServer) AddIdentityWithHistory(id *id.ID, validUntil,
	beginning time.Time, persistent bool,
	fallthroughProcessor message.Processor) {
	c.processor[*id] = fallthroughProcessor
}

func (c *mockCmixServer) AddIdentity(id *id.ID, validUntil time.Time,
	persistent bool, fallthroughProcessor message.Processor) {
	c.processor[*id] = fallthroughProcessor
}

func (c *mockCmixServer) RemoveIdentity(id *id.ID) {
	c.processor = nil
}

func (c *mockCmixServer) GetRoundResults(timeout time.Duration,
	roundCallback cmix.RoundEventCallback, roundList ...id.Round) {
	go func() {
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
	defer func() { c.curRnd += 1 }()

	rng := c.RNGStreamGenerator().GetStream()
	defer rng.Close()

	rnd := id.Round(c.curRnd)
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
			eId := receptionID.EphemeralIdentity{
				EphId:  ephIds[i],
				Source: recipients[i],
			}
			c.processor[*recipients[i]].Process(fmsg, nil, nil,
				eId, rounds.Round{ID: rnd})
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

type mockRPCServer struct {
	msgs chan format.Message
}

func (m *mockRPCServer) String() string {
	return "mockRPCServer"
}

func (m *mockRPCServer) Process(cMixMsg format.Message, _ []string, _ []byte,
	ephID receptionID.EphemeralIdentity, round rounds.Round) {
	m.msgs <- cMixMsg
}
