////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package auth

import (
	"bytes"
	"fmt"
	"sync"

	jww "github.com/spf13/jwalterweatherman"

	"gitlab.com/xx_network/primitives/id"

	"gitlab.com/elixxir/client/e2e/ratchet/partner/session"
	"gitlab.com/elixxir/client/interfaces/nike"
	"gitlab.com/elixxir/client/ratchet"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/ekv"
)

// AuthenticatedChannel
// TODO: Explain what an authenticated channel is and how it uses the
//       lower layers.
type AuthenticatedChannel struct {
	// The cMix ratchet.ID of the user who shares this authenticated channel.
	// Used for ratchet identification and debugging information.
	partner *id.ID
	// The cMix ID of the sender for this authenticated channel.
	// Used for ratchet identification and debugging information.
	me *id.ID

	ratchets ratchet.XXRatchet

	// Locks are handled on a per-ratchet basis
	// FIXME: this needs to be it's own primitive, because the map
	// accesses need their own lock. I couldn't find a kmutex
	// primitive that looked reasonable.. and only one was BSD...
	// I think we may have this pattern elsewhere, and we should add
	// a kmutex to primitives
	ratchetMuxes map[ratchet.ID]sync.Mutex
	stateMux     sync.Mutex

	// The key-value store for reading and writing to disk.
	kv ekv.KeyValue
	// params holds the ratcheting parameters
	params session.Params
	// cyHdlr holds callbacks for Adding and deleting
	// keys from the message processing layer.
	cyHdlr session.CypherHandler
}

// NewAuthenticatedChannel creates a fresh authenticated channel
// between 2 users.  This is called in case 1 and 2 above. Inside case
// 2, Alice would call SetState to update the state of the send
// ratchet to confirmed.
func NewAuthenticatedChannel(kv ekv.KeyValue, partner, me *id.ID, myPrivateKey nike.PrivateKey,
	myPublicKey, theirPublicKey nike.PublicKey, cypherHandler session.CypherHandler,
	params session.Params) *AuthenticatedChannel {
	senderID := makeRelationshipFingerprint(myPublicKey,
		theirPublicKey, me, partner)
	//	receiverID := makeRelationshipFingerprint(theirPublicKey,
	//		myPublicKey, partner, me)

	// FIXME: We need to crib the alg for determing the size of these bufs

	sendStates := make(map[ratchet.NegotiationState][]ratchet.ID)
	for i := 0; i < int(session.NewSessionCreated); i++ {
		sendStates[ratchet.NegotiationState(i)] = make([]ratchet.ID, 0)
	}

	sendStates[ratchet.NewSessionTriggered] = append(
		sendStates[ratchet.NewSessionTriggered], senderID)

	return &AuthenticatedChannel{
		me:      me,
		partner: partner,
		kv:      kv,
		params:  params,
		//		ratchetMuxes: ratchetLocks,
		cyHdlr: cypherHandler,
	}

}

// SetState performs the requested finite state conversion. It returns an error
// on an invalid conversion. It works by looking at the request state then
// searching the lists in sendStates to find the desired ID using the target
// newState to figure out where to look. It is called by any states where
// the transitions are explicitly defined above (3, 4, 7, 8)
func (ac *AuthenticatedChannel) SetState(senderID ratchet.ID,
	newState ratchet.NegotiationState) error {
	// TODO
	return nil
}

// SendRatchets returns the IDs of all of the active send ratchets.
// This is not thread safe. These IDs may not exist when you try to access them.
func (ac *AuthenticatedChannel) SendRatchets() []ratchet.ID {
	// TODO
	return nil
}

// SendRatchetsByState returns ids for all ratchets in the given state.
// This is not thread safe. These IDs may not exist when you try to access them.
func (ac *AuthenticatedChannel) SendRatchetsByState(
	state ratchet.NegotiationState) []ratchet.ID {
	/* FIXME
	ac.stateMux.Lock()
	defer ac.stateMux.Unlock()
	return ac.sendStates[state]
	*/
	return nil
}

// ReceiveRatchets returns the IDs of all of the active receive ratchets.
// This is not thread safe. These ratchet.IDs may not exist when you try to access them.
func (ac *AuthenticatedChannel) ReceiveRatchets() []ratchet.ID {
	// TODO
	return nil
}

// Encrypt selects an appropriate ratchet and uses it to encrypt data.
// If the ratchet is running out of keys, then it returns an error of type ???
// The caller should then also trigger a rekey.
// FIXME: Figure out the error type or how we are gonna do this?
func (ac *AuthenticatedChannel) Encrypt(plaintext []byte) (*ratchet.EncryptedMessage, error) {
	// TODO
	return nil, nil
}

// Decrypt finds the ratchet to decrypt with and decrypts an encrypted
// message. If the key has already been used, it refuses to do so and
// returns an error instead.
// NOTE: I think this could take a ratchet ratchet.ID instead of doing it's own lookup?
func (ac *AuthenticatedChannel) Decrypt(message *ratchet.EncryptedMessage) (plaintext []byte,
	err error) {
	return nil, nil
}

// TriggerRekey creates a new sending ratchet in the triggered state and returns
// the ratchet.ID and public key.
// This is called by case 5 above.
func (ac *AuthenticatedChannel) TriggerRekey() (ratchet.ID,
	nike.PublicKey) {
	// NOTE: The rekey trigger packet is of this form:
	// &RekeyTrigger{
	// 	PublicKey:   pubKey.Bytes(),
	// 	PqPublicKey: pqPubKeyBytes,
	// 	SessionID:   sess.GetSource().Marshal(),
	// })
	//
	// This is good, because we can send the SessionID of the most
	// recently used receive ratchet instead of the initial one, fixing a
	// fairly longstanding limitation in that we always DH off the original
	// key for the other party.
	return ratchet.ID{}, nil
}

// HandleRekeyTrigger creates a new receiving ratchet defined by the
// received rekey trigger public key.
// This is called by case 6 above.
// This calls the cyHdlr.AddKey() for each key fingerprint, and in theory can
// directly give it the Receive Ratchet, eliminating the need to even
// bother with a Decrypt function at this layer.
func (ac *AuthenticatedChannel) HandleRekeyTrigger(theirPublicKey nike.PublicKey) (ratchet.ID,
	nike.PublicKey) {
	return ratchet.ID{}, nil
}

// makeRelationshipFingerprint is copied from crypto/e2e/relationshipFingerprint
// and modified for the nike interface.
// creates a unique relationship fingerprint which can be used to ensure keys
// are unique and that message ratchet.IDs are unique
func makeRelationshipFingerprint(senderKey, receiverKey nike.PublicKey, sender,
	receiver *id.ID) ratchet.ID {
	h, err := hash.NewCMixHash()
	if err != nil {
		panic(fmt.Sprintf("Failed to get hash to make relationship"+
			" fingerprint with: %s", err))
	}

	senderKeyBytes := senderKey.Bytes()
	receiverKeyBytes := receiverKey.Bytes()

	switch bytes.Compare(senderKeyBytes, receiverKeyBytes) {
	case 1:
		h.Write(senderKeyBytes)
		h.Write(receiverKeyBytes)
	default:
		jww.WARN.Printf("Public keys the same, relationship " +
			"fingerproint uniqueness not assured")
		fallthrough
	case -1:
		h.Write(receiverKeyBytes)
		h.Write(senderKeyBytes)
	}

	id := ratchet.ID{}
	id.Unmarshal(h.Sum(nil))
	return id
}
