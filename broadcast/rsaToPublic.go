////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package broadcast

import (
	"encoding/binary"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/cmix/message"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	cryptoBroadcast "gitlab.com/elixxir/crypto/broadcast"
	"gitlab.com/elixxir/crypto/rsa"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
)

const (
	asymmetricRSAToPublicBroadcastServicePostfix = "AsymmToPublicBcast"
	asymmCMixSendTag                             = "AsymmetricBroadcast"
	internalPayloadSizeLength                    = 2
)

// BroadcastRSAtoPublic broadcasts the payload to the channel. Requires a
// healthy network state to send Payload length less than or equal to
// bc.MaxRSAToPublicPayloadSize, and the channel PrivateKey must be passed in
//
// BroadcastRSAtoPublic broadcasts the payload to the channel.
//
// The payload must be of the size [broadcastClient.MaxRSAToPublicPayloadSize]
// or smaller and the channel [rsa.PrivateKey] must be passed in.
//
// Tags are used to identity properties of the message for notifications
// For example, message types, senders, ect.
// The rate of false positives increases exponentially after more than
// 4 tags are used
//
// The network must be healthy to send. Returns [FreeNoAdminErr] if the
// channel disallows admin commands.
func (bc *broadcastClient) BroadcastRSAtoPublic(pk rsa.PrivateKey,
	payload []byte, tags []string, metadata [2]byte, cMixParams cmix.CMIXParams) (
	[]byte, rounds.Round, ephemeral.Id, error) {
	assemble := func(rid id.Round) ([]byte, error) { return payload, nil }
	return bc.BroadcastRSAToPublicWithAssembler(pk, assemble, tags, metadata, cMixParams)
}

// BroadcastRSAToPublicWithAssembler broadcasts the payload to the channel
// with a function that builds the payload based upon the ID of the selected
// round.
//
// The payload must be of the size [broadcastClient.MaxRSAToPublicPayloadSize]
// or smaller and the channel [rsa.PrivateKey] must be passed in.
//
// The network must be healthy to send. Returns [FreeNoAdminErr] if the
// channel disallows admin commands.
func (bc *broadcastClient) BroadcastRSAToPublicWithAssembler(
	pk rsa.PrivateKey, assembler Assembler, tags []string, metadata [2]byte,
	cMixParams cmix.CMIXParams) ([]byte, rounds.Round, ephemeral.Id, error) {
	if bc.Get().Options.AdminLevel == cryptoBroadcast.Announcement {
		return nil, rounds.Round{}, ephemeral.Id{}, FreeNoAdminErr
	}

	// Confirm network health
	if !bc.net.IsHealthy() {
		return nil, rounds.Round{}, ephemeral.Id{}, errors.New(errNetworkHealth)
	}

	var singleEncryptedPayload []byte
	assemble := func(rid id.Round) (fp format.Fingerprint,
		service cmix.Service, encryptedPayload, mac []byte, err error) {
		payload, err := assembler(rid)
		if err != nil {
			return format.Fingerprint{}, message.Service{}, nil,
				nil, err
		}
		// Check payload size
		if len(payload) > bc.MaxRSAToPublicPayloadSize() {
			return format.Fingerprint{}, message.Service{}, nil,
				nil, errors.Errorf(errPayloadSize, len(payload),
					bc.MaxRSAToPublicPayloadSize())
		}
		payloadLength := uint16(len(payload))

		finalPayload := make([]byte, bc.maxRSAToPublicPayloadSizeRaw())
		binary.BigEndian.PutUint16(finalPayload[:internalPayloadSizeLength],
			payloadLength)
		copy(finalPayload[internalPayloadSizeLength:], payload)

		// Encrypt payload
		singleEncryptedPayload, encryptedPayload, mac, fp, err =
			bc.channel.EncryptRSAToPublic(finalPayload, pk,
				bc.net.GetMaxMessageLength(), bc.rng.GetStream())
		if err != nil {
			return format.Fingerprint{}, message.Service{}, nil,
				nil, errors.WithMessage(err,
					"Failed to encrypt asymmetric broadcast message")
		}

		// Create service using asymmetric broadcast service tag and channel
		// reception ID allows anybody with this info to listen for messages on
		// this channel
		service = bc.GetRSAToPublicCompressedService(tags, metadata)

		if cMixParams.DebugTag == cmix.DefaultDebugTag {
			cMixParams.DebugTag = asymmCMixSendTag
		}

		// Create payload sized for sending over cmix
		sizedPayload := make([]byte, bc.net.GetMaxMessageLength())
		// Read random data into sized payload
		_, err = bc.rng.GetStream().Read(sizedPayload)
		if err != nil {
			return format.Fingerprint{}, message.Service{}, nil,
				nil, errors.WithMessage(err, "Failed to add "+
					"random data to sized broadcast")
		}
		copy(sizedPayload[:len(encryptedPayload)], encryptedPayload)

		return
	}

	r, ephID, err :=
		bc.net.SendWithAssembler(bc.channel.ReceptionID, assemble, cMixParams)
	return singleEncryptedPayload, r, ephID, err
}

func (bc *broadcastClient) GetRSAToPublicCompressedService(
	tags []string, metadata [2]byte) message.CompressedService {
	return message.CompressedService{
		Identifier: bc.asymIdentifier,
		Tags:       tags,
		Metadata:   metadata[:],
	}
}
