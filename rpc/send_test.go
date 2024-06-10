////////////////////////////////////////////////////////////////////////////////
// Copyright © 2024 xx foundation                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found         //
// in the LICENSE file                                                        //
////////////////////////////////////////////////////////////////////////////////

package rpc

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/cmix/message"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/nike/ecdh"
	"gitlab.com/xx_network/crypto/csprng"
)

func TestSend(t *testing.T) {
	rng := csprng.NewSystemRNG()
	serverID, err := generateRandomID(rng)
	require.NoError(t, err)

	serverPriv, serverPub := ecdh.ECDHNIKE.NewKeypair(rng)

}


type mockCmixServer struct {
	processor message.Processor
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
		result := make(map[id.Round]RoundResults)
		result[roundList[0]] = RoundResults {
			Status: cmix.Succeeded,
			Round: rounds.Round {
				ID: roundList[0],
			},
		}
		roundCallback(true, false, result)
	}()
}

func (c *mockCmixServer) RNGStreamGenerator() *fastRNG.StreamGenerator {
	s := fastRNG.NewStreamGenerator(1, 1, csprng.NewSystemRNG())
	return s
}

func (c *mockCmixServer) GetMaxMessageLength() int {
	return 0
}

func (c *mockCmixServer) SendManyWithAssembler(recipients []*id.ID,
	assembler cmix.ManyMessageAssembler, params cmix.CMIXParams) (
	rounds.Round, []ephemeral.Id, error) {
	return nil, nil nil
}
