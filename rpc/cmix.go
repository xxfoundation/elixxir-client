////////////////////////////////////////////////////////////////////////////////
// Copyright © 2024 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package rpc

import (
	"encoding/base64"
	"fmt"
	"io"
	"time"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/cmix/message"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/nike"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
)

// cMix functions required for this module.
type cMixClient interface {
	AddIdentityWithHistory(id *id.ID, validUntil, beginning time.Time,
		persistent bool, fallthroughProcessor message.Processor)
	AddIdentity(id *id.ID, validUntil time.Time, persistent bool,
		fallthroughProcessor message.Processor)
	RemoveIdentity(id *id.ID)
	GetRoundResults(timeout time.Duration,
		roundCallback cmix.RoundEventCallback, roundList ...id.Round)
	RNGStreamGenerator() *fastRNG.StreamGenerator
	GetMaxMessageLength() int
	SendManyWithAssembler(recipients []*id.ID, assembler cmix.ManyMessageAssembler,
		params cmix.CMIXParams) (rounds.Round, []ephemeral.Id, error)
}

func send(net cMixClient, recipient *id.ID, msg []byte,
	params cmix.CMIXParams) (rounds.Round, []ephemeral.Id, error) {
	if params.DebugTag == cmix.DefaultDebugTag {
		params.DebugTag = messageDebugTag
	}

	assemble := func(rid id.Round) ([]cmix.TargetedCmixMessage, error) {
		rng := net.RNGStreamGenerator().GetStream()
		defer rng.Close()

		payloadLen := maxPayloadLen(net)

		fpBytes, encryptedPayload, mac, err := createCMIXFields(
			msg, payloadLen, rng)
		if err != nil {
			return nil, err
		}

		fp := format.NewFingerprint(fpBytes)

		service := createRandomService(rng)

		sendMsg := cmix.TargetedCmixMessage{
			Recipient:   recipient,
			Payload:     encryptedPayload,
			Fingerprint: fp,
			Service:     service,
			Mac:         mac,
		}

		return []cmix.TargetedCmixMessage{sendMsg}, nil
	}
	return net.SendManyWithAssembler([]*id.ID{recipient}, assemble, params)
}

// Helper function that splits up the ciphertext into the appropriate cmix
// packet subfields
func createCMIXFields(ciphertext []byte, payloadSize int,
	rng io.Reader) (fpBytes, encryptedPayload, mac []byte, err error) {

	fpBytes = make([]byte, format.KeyFPLen)
	mac = make([]byte, format.MacLen)
	encryptedPayload = make([]byte, payloadSize-
		len(fpBytes)-len(mac)+2)

	// The first byte of mac and fp are random
	prefixBytes := make([]byte, 2)
	n, err := rng.Read(prefixBytes)
	if err != nil || n != len(prefixBytes) {
		err = fmt.Errorf("rng read failure: %+v", err)
		return nil, nil, nil, err
	}
	// Note: the first bit must be 0 for these...
	fpBytes[0] = 0x7F & prefixBytes[0]
	mac[0] = 0x7F & prefixBytes[1]

	// ciphertext[0:FPLen-1] == fp[1:FPLen]
	start := 0
	end := format.KeyFPLen - 1
	copy(fpBytes[1:format.KeyFPLen], ciphertext[start:end])
	// ciphertext[FPLen-1:FPLen+MacLen-2] == mac[1:MacLen]
	start = end
	end = start + format.MacLen - 1
	copy(mac[1:format.MacLen], ciphertext[start:end])
	// ciphertext[FPLen+MacLen-2:] == encryptedPayload
	start = end
	end = start + len(encryptedPayload)
	copy(encryptedPayload, ciphertext[start:end])

	// Fill anything left w/ random data
	numLeft := end - start - len(encryptedPayload)
	if numLeft > 0 {
		jww.WARN.Printf("[DM] small ciphertext, added %d bytes",
			numLeft)
		n, err := rng.Read(encryptedPayload[end-start:])
		if err != nil || n != numLeft {
			err = fmt.Errorf("rng read failure: %+v", err)
			return nil, nil, nil, err
		}
	}

	return fpBytes, encryptedPayload, mac, nil
}

func createRandomService(rng io.Reader) message.Service {
	// NOTE: 64 is entirely arbitrary, 33 bytes are used for the ID
	// and the rest will be base64'd into a string for the tag.
	data := make([]byte, 64)
	n, err := rng.Read(data)
	if err != nil {
		jww.FATAL.Panicf("rng failure: %+v", err)
	}
	if n != len(data) {
		jww.FATAL.Panicf("rng read failure, short read: %d < %d", n,
			len(data))
	}
	return message.Service{
		Identifier: data[:33],
		Tag:        base64.RawStdEncoding.EncodeToString(data[33:]),
	}
}

func generateRandomKeys(scheme nike.Nike, rng io.Reader) (nike.PrivateKey,
	nike.PublicKey, error) {
	// Create compatible keypair
	privateKeyBytes := make([]byte, scheme.PrivateKeySize())
	n, err := rng.Read(privateKeyBytes)
	if err != nil {
		return nil, nil, err
	}
	if n != len(privateKeyBytes) {
		return nil, nil, errors.Errorf("short read: %d < %d", n,
			len(privateKeyBytes))
	}

	privateKey := scheme.NewEmptyPrivateKey()
	err = privateKey.FromBytes(privateKeyBytes)
	if err != nil {
		return nil, nil, err
	}
	publicKey := scheme.DerivePublicKey(privateKey)

	return privateKey, publicKey, nil
}

func generateRandomID(rng io.Reader) (*id.ID, error) {
	idBytes := make([]byte, id.ArrIDLen)
	n, err := rng.Read(idBytes)
	if err != nil {
		return nil, err
	}
	if n != len(idBytes) {
		return nil, errors.Errorf("short read, %d < %d",
			n, len(idBytes))
	}
	newID := &id.ID{}
	copy(newID[:], idBytes)
	newID.SetType(id.User)
	return newID, nil
}

func maxPayloadLen(net cMixClient) int {
	// As we don't use the mac or fp fields, we can extend
	// our payload size
	// (-2 to eliminate the first byte of mac and fp)
	return net.GetMaxMessageLength() +
		format.MacLen + format.KeyFPLen - 2

}
