///////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"encoding/json"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/broadcast"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	cryptoBroadcast "gitlab.com/elixxir/crypto/broadcast"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id/ephemeral"
)

// Channel is a bindings-level struct encapsulating the broadcast.Channel client object.
type Channel struct {
	ch broadcast.Channel
}

// ChannelDef is the bindings representation of an elixxir/crypto broadcast.Channel object.
//
// Example JSON:
//  {"Name": "My broadcast channel",
//   "Description":"A broadcast channel for me to test things",
//   "Salt":"gpUqW7N22sffMXsvPLE7BA==",
//   "PubKey":"LS0tLS1CRUdJTiBSU0EgUFVCTElDIEtFWS0tLS0tCk1DZ0NJUUN2YkZVckJKRFpqT3Y0Y0MvUHZZdXNvQkFtUTFkb3Znb044aHRuUjA2T3F3SURBUUFCCi0tLS0tRU5EIFJTQSBQVUJMSUMgS0VZLS0tLS0="
//  }
type ChannelDef struct {
	Name        string
	Description string
	Salt        []byte
	PubKey      []byte
}

// BroadcastMessage is the bindings representation of a broadcast message.
//
// Example JSON:
//  {"RoundID":42,
//   "EphID":[0,0,0,0,0,0,24,61],
//   "Payload":"SGVsbG8sIGJyb2FkY2FzdCBmcmllbmRzIQ=="
//  }
type BroadcastMessage struct {
	BroadcastReport
	Payload []byte
}

// BroadcastReport is the bindings representation of the info on how a broadcast message was sent
//
// Example JSON:
//  {"RoundID":42,
//   "EphID":[0,0,0,0,0,0,24,61]
//  }
type BroadcastReport struct {
	RoundsList
	EphID ephemeral.Id
}

// BroadcastListener is the public function type bindings can use to listen for
// broadcast messages.
//
// Parameters:
//  - []byte - the JSON marshalled bytes of the BroadcastMessage object, which
//    can be passed into WaitForRoundResult to see if the broadcast succeeded.
type BroadcastListener interface {
	Callback([]byte, error)
}

// NewBroadcastChannel creates a bindings-layer broadcast channel & starts listening for new messages
//
// Parameters:
//  - cmixId - internal ID of cmix
//  - channelDefinition - JSON marshalled ChannelDef object
func NewBroadcastChannel(cmixId int, channelDefinition []byte) (*Channel, error) {
	c, err := cmixTrackerSingleton.get(cmixId)
	if err != nil {
		return nil, err
	}

	def := &ChannelDef{}
	err = json.Unmarshal(channelDefinition, def)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to unmarshal underlying channel definition")
	}

	channelID, err := cryptoBroadcast.NewChannelID(def.Name, def.Description, def.Salt, def.PubKey)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to generate channel ID")
	}
	chanPubLoaded, err := rsa.LoadPublicKeyFromPem(def.PubKey)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to load public key")
	}

	ch, err := broadcast.NewBroadcastChannel(cryptoBroadcast.Channel{
		ReceptionID: channelID,
		Name:        def.Name,
		Description: def.Description,
		Salt:        def.Salt,
		RsaPubKey:   chanPubLoaded,
	}, c.api.GetCmix(), c.api.GetRng())
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to create broadcast channel client")
	}

	return &Channel{ch: ch}, nil
}

// Listen registers a BroadcastListener for a given method.
// This allows users to handle incoming broadcast messages.
//
// Parameters:
//  - l - BroadcastListener object
//  - method - int corresponding to broadcast.Method constant, 0 for symmetric
//    or 1 for asymmetric
func (c *Channel) Listen(l BroadcastListener, method int) error {
	broadcastMethod := broadcast.Method(method)
	listen := func(payload []byte,
		receptionID receptionID.EphemeralIdentity, round rounds.Round) {
		l.Callback(json.Marshal(&BroadcastMessage{
			BroadcastReport: BroadcastReport{
				RoundsList: makeRoundsList(round.ID),
				EphID:      receptionID.EphId,
			},
			Payload: payload,
		}))
	}
	return c.ch.RegisterListener(listen, broadcastMethod)
}

// Broadcast sends a given payload over the broadcast channel using symmetric
// broadcast.
//
// Returns:
//  - []byte - the JSON marshalled bytes of the BroadcastReport object, which
//    can be passed into WaitForRoundResult to see if the broadcast succeeded.
func (c *Channel) Broadcast(payload []byte) ([]byte, error) {
	rid, eid, err := c.ch.Broadcast(payload, cmix.GetDefaultCMIXParams())
	if err != nil {
		return nil, err
	}
	return json.Marshal(BroadcastReport{
		RoundsList: makeRoundsList(rid),
		EphID:      eid,
	})
}

// BroadcastAsymmetric sends a given payload over the broadcast channel using
// asymmetric broadcast. This mode of encryption requires a private key.
//
// Returns:
//  - []byte - the JSON marshalled bytes of the BroadcastReport object, which
//    can be passed into WaitForRoundResult to see if the broadcast succeeded.
func (c *Channel) BroadcastAsymmetric(payload, pk []byte) ([]byte, error) {
	pkLoaded, err := rsa.LoadPrivateKeyFromPem(pk)
	if err != nil {
		return nil, err
	}
	rid, eid, err := c.ch.BroadcastAsymmetric(pkLoaded, payload, cmix.GetDefaultCMIXParams())
	if err != nil {
		return nil, err
	}
	return json.Marshal(BroadcastReport{
		RoundsList: makeRoundsList(rid),
		EphID:      eid,
	})
}

// MaxPayloadSize returns the maximum possible payload size which can be broadcast.
func (c *Channel) MaxPayloadSize() int {
	return c.ch.MaxPayloadSize()
}

// MaxAsymmetricPayloadSize returns the maximum possible payload size which can be broadcast.
func (c *Channel) MaxAsymmetricPayloadSize() int {
	return c.ch.MaxAsymmetricPayloadSize()
}

// Get returns the result of calling json.Marshal on a ChannelDef based on the underlying crypto broadcast.Channel.
func (c *Channel) Get() ([]byte, error) {
	def := c.ch.Get()
	return json.Marshal(&ChannelDef{
		Name:        def.Name,
		Description: def.Description,
		Salt:        def.Salt,
		PubKey:      rsa.CreatePublicKeyPem(def.RsaPubKey),
	})
}

// Stop stops the channel from listening for more messages.
func (c *Channel) Stop() {
	c.ch.Stop()
}
