////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"crypto/ed25519"
	"fmt"
	"time"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/crypto/channel"
	cryptoChannel "gitlab.com/elixxir/crypto/channel"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/netTime"
	"golang.org/x/crypto/blake2b"
	"google.golang.org/protobuf/proto"

	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v4/cmix/message"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/crypto/dm"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/nike"
	"gitlab.com/elixxir/crypto/nike/ecdh"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/states"
)

const (
	dmStoreKey              = "dmToken-%s"
	dmStoreVersion          = 0
	directMessageServiceTag = "direct_message_v0"
)

// DeriveReceptionID returns a reception ID for direct messages sent
// to the user. It generates this ID by hashing the public key and
// an arbitrary idToken together. The ID type is set to "User".
func DeriveReceptionID(publicKey nike.PublicKey, idToken []byte) *id.ID {
	h, err := blake2b.New256(nil)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	h.Write(publicKey.Bytes())
	h.Write(idToken)
	idBytes := h.Sum(nil)
	idBytes = append(idBytes, byte(id.User))
	receptionID, err := id.Unmarshal(idBytes)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	return receptionID
}

type dmClient struct {
	receptionID *id.ID
	privateKey  nike.PrivateKey
	publicKey   nike.PublicKey
	myToken     []byte

	nm  *nicknameManager
	net Client
	rng *fastRNG.StreamGenerator
}

// NewDMClient creates a new client for direct messaging. This should
// be called when the channels manager is created/loaded. It has no
// associated state, so it does not have a corresponding Load
// funciton.
func NewDMClient(privateKey nike.PrivateKey,
	myIdToken []byte, nickMgr *nicknameManager,
	net Client,
	rng *fastRNG.StreamGenerator) *dmClient {

	publicKey := ecdh.ECDHNIKE.DerivePublicKey(privateKey)

	receptionID := DeriveReceptionID(publicKey, myIdToken)

	// TODO: do we do the reception registration here or do we do
	// it outside of this function?
	// similarly, do we take a full cMix client or do we
	// just take a sender function? I think I favor the latter...

	return &dmClient{
		receptionID: receptionID,
		privateKey:  privateKey,
		publicKey:   publicKey,
		nm:          nickMgr,
		net:         net,
		rng:         rng,
	}
}

// GetDMNIKEPublicKey converts a public key from a signing key to a NIKE
// (key exchange compatible) version of the the public key.
func GetDMNIKEPublicKey(publicEdwardsKey *ed25519.PublicKey) nike.PublicKey {
	publicKey := ecdh.ECDHNIKE.NewEmptyPublicKey()
	publicKey.(*ecdh.PublicKey).FromEdwards(*publicEdwardsKey)
	return publicKey
}

// GetDMNIKEPrivateKey converts a private key from a signing key to a NIKE
// (key exchange compatible) version of the the private key.
func GetDMNIKEPrivateKey(privateEdwardsKey *ed25519.PrivateKey) nike.PrivateKey {
	privateKey := ecdh.ECDHNIKE.NewEmptyPrivateKey()
	privateKey.(*ecdh.PrivateKey).FromEdwards(*privateEdwardsKey)
	return privateKey
}

func (dc *dmClient) Send(partnerToken []byte, partnerPubKey nike.PublicKey,
	messageType MessageType, msg []byte, params cmix.CMIXParams) (
	cryptoChannel.MessageID, rounds.Round, ephemeral.Id, error) {

	if params.DebugTag == cmix.DefaultDebugTag {
		params.DebugTag = directMessageServiceTag
	}

	partnerID := DeriveReceptionID(partnerPubKey, partnerToken)

	// Note: We log sends on exit, and append what happened to the message
	// this cuts down on clutter in the log.
	sendPrint := fmt.Sprintf("[%s] Sending dm to %s type %d at %s",
		params.DebugTag, partnerID, messageType, netTime.Now())
	defer jww.INFO.Println(sendPrint)

	nickname, _ := dc.nm.GetNickname(partnerID)

	var msgId cryptoChannel.MessageID

	// Generate random nonce to be used for message ID
	// generation. This makes it so two identical messages sent on
	// the same round have different message IDs.
	rng := dc.rng.GetStream()
	msgNonce := make([]byte, messageNonceSize)
	n, err := rng.Read(msgNonce)
	rng.Close()
	if err != nil {
		sendPrint += fmt.Sprintf(", failed to generate nonce: %+v", err)
		return cryptoChannel.MessageID{}, rounds.Round{},
			ephemeral.Id{},
			errors.Errorf("Failed to generate nonce: %+v", err)
	} else if n != messageNonceSize {
		sendPrint += fmt.Sprintf(
			", got %d bytes for %d-byte nonce", n, messageNonceSize)
		return cryptoChannel.MessageID{}, rounds.Round{},
			ephemeral.Id{},
			errors.Errorf(
				"Generated %d bytes for %d-byte nonce", n,
				messageNonceSize)
	}

	directMessage := &DirectMessage{
		Lease:          0,
		ECCPublicKey:   dc.publicKey.Bytes(),
		DMToken:        dc.myToken,
		PayloadType:    uint32(messageType),
		Payload:        msg,
		Nickname:       nickname,
		Nonce:          msgNonce,
		LocalTimestamp: netTime.Now().UnixNano(),
	}

	// Send to Partner
	assemble := func(rid id.Round) (fp format.Fingerprint,
		service message.Service, encryptedPayload, mac []byte,
		err error) {

		directMessage.RoundID = uint64(rid)

		// Serialize the message
		dmSerial, err := proto.Marshal(directMessage)
		if err != nil {
			return
		}

		// Make the messageID
		msgId = cryptoChannel.MakeMessageID(dmSerial, &id.DummyUser)

		// NOTE: When sending you use the partner id
		//       When self sending you use your own id
		//       Receiver figures out what to do based on msg content
		service = message.Service{
			Identifier: partnerID.Bytes(),
			Tag:        directMessageServiceTag,
		}

		// As we don't use the mac or fp fields, we can extend
		// our payload size
		// (-2 to eliminate the first byte of mac and fp)
		payloadLen := (dc.net.GetMaxMessageLength() +
			format.MacLen + format.KeyFPLen - 2)

		ciphertext := dm.Cipher.Encrypt(dmSerial, partnerPubKey,
			payloadLen)

		fpBytes, encryptedPayload, mac, err := dc.createCMIXFields(
			ciphertext)

		fp = format.NewFingerprint(fpBytes)

		return fp, service, encryptedPayload, mac, err
	}
	partnerRnd, partnerEphID, err := dc.net.SendWithAssembler(partnerID,
		assemble,
		params)
	if err != nil {
		sendPrint += fmt.Sprintf(", err on partner send: %+v", err)
	}
	sendPrint += fmt.Sprintf(", partner send eph %s rnd %s id %s",
		partnerEphID, partnerRnd.ID, msgId)

	// SELF Send, NOTE: This is the send that returns the message ID
	// for tracking. We can't track the receipt of the DM because we
	// never pick it up.
	selfAssemble := func(rid id.Round) (fp format.Fingerprint,
		service message.Service, encryptedPayload, mac []byte,
		err error) {

		directMessage.RoundID = uint64(rid)
		// NOTE: Very important to overwrite these fields
		// for self sending!
		directMessage.ECCPublicKey = partnerPubKey.Bytes()
		directMessage.DMToken = partnerToken

		// Serialize the message
		dmSerial, err := proto.Marshal(directMessage)
		if err != nil {
			return
		}

		// Make the messageID
		msgId = cryptoChannel.MakeMessageID(dmSerial, &id.DummyUser)

		// NOTE: When sending you use the partner id
		//       When self sending you use your own id
		//       Receiver figures out what to do based on msg content
		service = message.Service{
			Identifier: dc.receptionID.Bytes(),
			Tag:        directMessageServiceTag,
		}

		if params.DebugTag == cmix.DefaultDebugTag {
			params.DebugTag = directMessageServiceTag
		}

		// As we don't use the mac or fp fields, we can extend
		// our payload size
		// (-2 to eliminate the first byte of mac and fp)
		payloadLen := (dc.net.GetMaxMessageLength() +
			format.MacLen + format.KeyFPLen - 2)

		// FIXME: Why does this one return an error when the
		// other doesn't!?
		ciphertext, err := dm.Cipher.EncryptSelf(dmSerial, dc.privateKey,
			payloadLen)
		if err != nil {
			return
		}

		fpBytes, encryptedPayload, mac, err := dc.createCMIXFields(
			ciphertext)

		fp = format.NewFingerprint(fpBytes)

		return fp, service, encryptedPayload, mac, err
	}
	myRnd, myEphID, err := dc.net.SendWithAssembler(dc.receptionID,
		selfAssemble,
		params)
	if err != nil {
		sendPrint += fmt.Sprintf(", err on self send: %+v", err)
	}
	sendPrint += fmt.Sprintf(", self send eph %s rnd %s id %s",
		myEphID, myRnd.ID, msgId)

	return msgId, myRnd, myEphID, err

}

// RegisterListener registers a listener for broadcast messages.
func (dc *dmClient) RegisterListener(listenerCb ListenerFunc,
	checkSent messageReceiveFunc) error {
	p := &processor{
		c:         dc,
		cb:        listenerCb,
		checkSent: checkSent,
	}

	service := message.Service{
		Identifier: dc.receptionID.Bytes(),
		Tag:        directMessageServiceTag,
	}

	dc.net.AddService(dc.receptionID, service, p)
	return nil
}

// ListenerFunc is registered when creating a new dm listener and
// receives all new messages for the reception ID.
type ListenerFunc func(msgID channel.MessageID, dmi *DirectMessage,
	ts time.Time, ephID receptionID.EphemeralIdentity, round rounds.Round,
	status SentStatus) (uint64, error)

// processor struct for message handling
type processor struct {
	c         *dmClient
	cb        ListenerFunc
	checkSent messageReceiveFunc
}

// String returns a string identifying the symmetricProcessor for
// debugging purposes.
func (p *processor) String() string {
	return "directMessage-"
}

// Process decrypts the broadcast message and sends the results on the callback.
func (p *processor) Process(msg format.Message,
	receptionID receptionID.EphemeralIdentity, round rounds.Round) {

	ciphertext := reconstructCiphertext(msg)

	var payload []byte
	var err error
	if dm.Cipher.IsSelfEncrypted(ciphertext, p.c.privateKey) {
		payload, err = dm.Cipher.DecryptSelf(ciphertext,
			p.c.privateKey)
	} else {
		payload, err = dm.Cipher.Decrypt(ciphertext,
			p.c.privateKey)
	}
	if err != nil {
		jww.ERROR.Printf("failed to decrypt direct message: %s", err)
		return
	}

	msgID := channel.MakeMessageID(payload, &id.DummyUser)

	// Check if we sent the message and ignore triggering if we sent
	if p.checkSent(msgID, round) {
		return
	}

	directMsg := &DirectMessage{}
	if err := proto.Unmarshal(payload, directMsg); err != nil {
		jww.ERROR.Printf("unable to parse direct message: %+v",
			err)
		return
	}

	/* CRYPTOGRAPHICALLY RELEVANT CHECKS */

	// FIXME: MsgID on a send cannot match MsgID on a self send..
	//        it's not clear how to fix that.. Maybe we need
	//        to set the round id to the same on both send and self
	//        send, and make this check sensitive to that?
	//        We will also have to either wipe or change the
	//        ECC PubKey + dmToken fields to make it work, and that
	//        means reserializing the messge :-(

	// Check the round to ensure the message is not a replay
	if id.Round(directMsg.RoundID) != round.ID {
		jww.WARN.Printf("The round DM %s send on %d"+
			"by %s was not the same as the"+
			"round the message was found on (%d)", msgID,
			round.ID, directMsg.ECCPublicKey, directMsg.RoundID)
		return
	}

	// NOTE: There's no signature here, that kind of thing is done
	// by Noise in the layer doing decryption.
	//
	// There also are no admin commands for direct messages.

	// Replace the timestamp on the message if it is outside the
	// allowable range
	ts := vetTimestamp(time.Unix(0, directMsg.LocalTimestamp),
		round.Timestamps[states.QUEUED], msgID)

	p.cb(msgID, directMsg, ts, receptionID, round, Delivered)
}

// enableDirectMessageToken is a helper functions for EnableDirectMessageToken
// which directly sets a token for the given channel ID into storage. This is an
// unsafe operation.
func (m *manager) enableDirectMessageToken(chId *id.ID) error {
	privKey := m.me.Privkey
	toStore := hashPrivateKey(privKey)
	vo := &versioned.Object{
		Version:   dmStoreVersion,
		Timestamp: netTime.Now(),
		Data:      toStore,
	}

	return m.kv.Set(createDmStoreKey(chId), vo)

}

// disableDirectMessageToken is a helper functions for DisableDirectMessageToken
// which deletes a token for the given channel ID into storage. This is an
// unsafe operation.
func (m *manager) disableDirectMessageToken(chId *id.ID) error {
	return m.kv.Delete(createDmStoreKey(chId), dmStoreVersion)
}

// getDmToken will retrieve a DM token from storage. If EnableDirectMessageToken
// has been called on this channel, then a token will exist in storage and be
// returned. If EnableDirectMessageToken has not been called on this channel,
// no token will exist and getDmToken will return nil.
func (m *manager) getDmToken(chId *id.ID) []byte {
	obj, err := m.kv.Get(createDmStoreKey(chId), dmStoreVersion)
	if err != nil {
		return nil
	}
	return obj.Data
}

func createDmStoreKey(chId *id.ID) string {
	return fmt.Sprintf(dmStoreKey, chId)

}

// hashPrivateKey is a helper function which generates a DM token.
// As per spec, this is just a hash of the private key.
func hashPrivateKey(privKey *ed25519.PrivateKey) []byte {
	h, err := hash.NewCMixHash()
	if err != nil {
		jww.FATAL.Panicf("Failed to generate cMix hash: %+v", err)
	}

	h.Write(privKey.Seed())
	return h.Sum(nil)
}

// Helper function that splits up the ciphertext into the appropriate cmix
// packet subfields
func (dc *dmClient) createCMIXFields(ciphertext []byte) (fpBytes,
	encryptedPayload, mac []byte, err error) {

	rng := dc.rng.GetStream()
	defer rng.Close()

	fpBytes = make([]byte, format.KeyFPLen)
	mac = make([]byte, format.MacLen)
	encryptedPayload = make([]byte, dc.net.GetMaxMessageLength())

	// The first byte of mac and fp are random
	prefixBytes := make([]byte, 2)
	n, err := rng.Read(prefixBytes)
	if err != nil || n != len(prefixBytes) {
		err = fmt.Errorf("rng read failure: %+v", err)
		return nil, nil, nil, err
	}
	fpBytes[0] = prefixBytes[0]
	mac[0] = prefixBytes[1]

	// ciphertext[0:FPLen-1] == fp[1:FPLen]
	start := 0
	end := format.KeyFPLen
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

// This helper does the opposite of "createCMIXFields" above
func reconstructCiphertext(msg format.Message) []byte {
	fp := msg.GetKeyFP()
	res := fp[1:]
	res = append(res, msg.GetMac()[1:]...)
	res = append(res, msg.GetContents()...)
	return res
}
