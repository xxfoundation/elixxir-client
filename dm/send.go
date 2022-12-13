////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package dm

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/cmix/message"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/client/v4/emoji"
	"gitlab.com/elixxir/crypto/dm"
	cryptoMessage "gitlab.com/elixxir/crypto/message"
	"gitlab.com/elixxir/crypto/nike"
	"gitlab.com/elixxir/crypto/nike/ecdh"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/netTime"
	"golang.org/x/crypto/blake2b"
	"google.golang.org/protobuf/proto"
)

const (
	textVersion     = 0
	reactionVersion = 0

	// SendMessageTag is the base tag used when generating a debug tag for
	// sending a message.
	SendMessageTag = "Message"

	// SendReplyTag is the base tag used when generating a debug tag for
	// sending a reply.
	SendReplyTag = "Reply"

	// SendReactionTag is the base tag used when generating a debug tag for
	// sending a reaction.
	SendReactionTag = "Reaction"

	directMessageServiceTag = "direct_message_v0"
	// The size of the nonce used in the message ID.
	messageNonceSize = 4
)

// SendText is used to send a formatted message to another user.
func (dc *dmClient) SendText(partnerPubKey *ed25519.PublicKey, partnerToken uint32,
	msg string, params cmix.CMIXParams) (
	cryptoMessage.ID, rounds.Round, ephemeral.Id, error) {
	return dc.SendReply(partnerPubKey, partnerToken, msg,
		cryptoMessage.ID{}, params)
}

// SendDMReply is used to send a formatted direct message reply.
//
// If the message ID that the reply is sent to does not exist,
// then the other side will post the message as a normal
// message and not as a reply.
func (dc *dmClient) SendReply(partnerPubKey *ed25519.PublicKey,
	partnerToken uint32, msg string, replyTo cryptoMessage.ID,
	params cmix.CMIXParams) (cryptoMessage.ID, rounds.Round, ephemeral.Id, error) {

	tag := makeDebugTag(*partnerPubKey, []byte(msg), SendReplyTag)
	jww.INFO.Printf("[%s]SendReply(%s, to %s)", tag, partnerPubKey, replyTo)
	txt := &Text{
		Version:        textVersion,
		Text:           msg,
		ReplyMessageID: replyTo[:],
	}

	params = params.SetDebugTag(tag)

	txtMarshaled, err := proto.Marshal(txt)
	if err != nil {
		return cryptoMessage.ID{}, rounds.Round{},
			ephemeral.Id{}, err
	}

	return dc.Send(partnerPubKey, partnerToken, TextType, txtMarshaled,
		params)
}

// SendReaction is used to send a reaction to a direct
// message. The reaction must be a single emoji with no other
// characters, and will be rejected otherwise.
//
// Clients will drop the reaction if they do not recognize the reactTo
// message.
func (dc *dmClient) SendReaction(partnerPubKey *ed25519.PublicKey,
	partnerToken uint32, reaction string, reactTo cryptoMessage.ID,
	params cmix.CMIXParams) (cryptoMessage.ID,
	rounds.Round, ephemeral.Id, error) {
	tag := makeDebugTag(*partnerPubKey, []byte(reaction),
		SendReactionTag)
	jww.INFO.Printf("[%s]SendReply(%s, to %s)", tag, *partnerPubKey,
		reactTo)

	if err := emoji.ValidateReaction(reaction); err != nil {
		return cryptoMessage.ID{}, rounds.Round{}, ephemeral.Id{}, err
	}

	react := &Reaction{
		Version:           reactionVersion,
		Reaction:          reaction,
		ReactionMessageID: reactTo[:],
	}

	params = params.SetDebugTag(tag)

	reactMarshaled, err := proto.Marshal(react)
	if err != nil {
		return cryptoMessage.ID{}, rounds.Round{}, ephemeral.Id{}, err
	}

	return dc.Send(partnerPubKey, partnerToken, ReactionType,
		reactMarshaled, params)
}

func (dc *dmClient) Send(partnerEdwardsPubKey *ed25519.PublicKey,
	partnerToken uint32, messageType MessageType, msg []byte,
	params cmix.CMIXParams) (
	cryptoMessage.ID, rounds.Round, ephemeral.Id, error) {

	partnerPubKey := ecdh.Edwards2ECDHNIKEPublicKey(partnerEdwardsPubKey)

	partnerID := deriveReceptionID(partnerPubKey, partnerToken)

	// Note: We log sends on exit, and append what happened to the message
	// this cuts down on clutter in the log.
	sendPrint := fmt.Sprintf("[%s] Sending dm to %s type %d at %s",
		params.DebugTag, partnerID, messageType, netTime.Now())
	defer jww.INFO.Println(sendPrint)

	rng := dc.rng.GetStream()
	defer rng.Close()

	nickname, _ := dc.nm.GetNickname(partnerID)

	// Generate random nonce to be used for message ID
	// generation. This makes it so two identical messages sent on
	// the same round have different message IDs.
	msgNonce := make([]byte, messageNonceSize)
	n, err := rng.Read(msgNonce)
	if err != nil {
		sendPrint += fmt.Sprintf(", failed to generate nonce: %+v", err)
		return cryptoMessage.ID{}, rounds.Round{},
			ephemeral.Id{},
			errors.Errorf("Failed to generate nonce: %+v", err)
	} else if n != messageNonceSize {
		sendPrint += fmt.Sprintf(
			", got %d bytes for %d-byte nonce", n, messageNonceSize)
		return cryptoMessage.ID{}, rounds.Round{},
			ephemeral.Id{},
			errors.Errorf(
				"Generated %d bytes for %d-byte nonce", n,
				messageNonceSize)
	}

	directMessage := &DirectMessage{
		DMToken:        dc.myToken,
		PayloadType:    uint32(messageType),
		Payload:        msg,
		Nickname:       nickname,
		Nonce:          msgNonce,
		LocalTimestamp: netTime.Now().UnixNano(),
	}

	msgID := cryptoMessage.DeriveDirectMessageID(partnerID,
		directMessage)

	if params.DebugTag == cmix.DefaultDebugTag {
		params.DebugTag = directMessageServiceTag
	}
	partnerRnd, partnerEphID, err := send(dc.net, partnerID, partnerPubKey,
		dc.privateKey, directMessage, params, rng)
	if err != nil {
		sendPrint += fmt.Sprintf(", err on partner send: %+v", err)
		return cryptoMessage.ID{}, rounds.Round{},
			ephemeral.Id{}, err
	}
	sendPrint += fmt.Sprintf(", partner send eph %s rnd %s id %s",
		partnerEphID, partnerRnd.ID, msgID)

	myRnd, myEphID, err := sendSelf(dc.net, dc.receptionID, partnerPubKey,
		partnerToken, dc.privateKey, directMessage, params, rng)
	if err != nil {
		sendPrint += fmt.Sprintf(", err on self send: %+v", err)
		return cryptoMessage.ID{}, rounds.Round{},
			ephemeral.Id{}, err
	}
	sendPrint += fmt.Sprintf(", self send eph %s rnd %s id %s",
		myEphID, myRnd.ID, msgID)

	return msgID, myRnd, myEphID, err

}

// DeriveReceptionID returns a reception ID for direct messages sent
// to the user. It generates this ID by hashing the public key and
// an arbitrary idToken together. The ID type is set to "User".
func DeriveReceptionID(publicKey ed25519.PublicKey, idToken uint32) *id.ID {
	nikePubKey := ecdh.Edwards2ECDHNIKEPublicKey(&publicKey)
	return deriveReceptionID(nikePubKey, idToken)
}

func deriveReceptionID(publicKey nike.PublicKey, idToken uint32) *id.ID {
	h, err := blake2b.New256(nil)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	h.Write(publicKey.Bytes())
	tokenBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(tokenBytes, idToken)
	h.Write(tokenBytes)
	idBytes := h.Sum(nil)
	idBytes = append(idBytes, byte(id.User))
	receptionID, err := id.Unmarshal(idBytes)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	return receptionID
}

func send(net cMixClient, partnerID *id.ID, partnerPubKey nike.PublicKey,
	myPrivateKey nike.PrivateKey,
	msg *DirectMessage, params cmix.CMIXParams, rng io.Reader) (rounds.Round,
	ephemeral.Id, error) {

	// Send to Partner
	assemble := func(rid id.Round) (fp format.Fingerprint,
		service message.Service, encryptedPayload, mac []byte,
		err error) {

		msg.RoundID = uint64(rid)

		// Serialize the message
		dmSerial, err := proto.Marshal(msg)
		if err != nil {
			return
		}

		// NOTE: When sending you use the partner id
		//       When self sending you use your own id
		//       Receiver figures out what to do based on msg content
		service = message.Service{
			Identifier: partnerID.Bytes(),
			Tag:        directMessageServiceTag,
		}

		jww.INFO.Printf("[DM] Payload In: \n%v", dmSerial)

		payloadLen := calcDMPayloadLen(net)

		ciphertext := dm.Cipher.Encrypt(dmSerial, myPrivateKey,
			partnerPubKey, rng, payloadLen)

		fpBytes, encryptedPayload, mac, err := createCMIXFields(
			ciphertext, payloadLen, rng)

		fp = format.NewFingerprint(fpBytes)

		return fp, service, encryptedPayload, mac, err
	}
	return net.SendWithAssembler(partnerID, assemble, params)
}

func sendSelf(net cMixClient, myID *id.ID, partnerPubKey nike.PublicKey,
	partnerToken uint32, myPrivateKey nike.PrivateKey,
	msg *DirectMessage, params cmix.CMIXParams, rng io.Reader) (rounds.Round,
	ephemeral.Id, error) {

	// SELF Send, NOTE: This is the send that returns the message ID
	// for tracking. We can't track the receipt of the DM because we
	// never pick it up.
	selfAssemble := func(rid id.Round) (fp format.Fingerprint,
		service message.Service, encryptedPayload, mac []byte,
		err error) {

		// NOTE: We do not modify the round id already in the
		//       message object. This enables the same msgID
		//       on sender and recipient.
		msg.SelfRoundID = uint64(rid)
		// NOTE: Very important to overwrite these fields
		// for self sending!
		msg.DMToken = partnerToken

		// Serialize the message
		dmSerial, err := proto.Marshal(msg)
		if err != nil {
			return
		}

		// NOTE: When sending you use the partner id
		//       When self sending you use your own id
		//       Receiver figures out what to do based on msg content
		service = message.Service{
			Identifier: myID.Bytes(),
			Tag:        directMessageServiceTag,
		}

		if params.DebugTag == cmix.DefaultDebugTag {
			params.DebugTag = directMessageServiceTag
		}

		jww.INFO.Printf("[DM] SelfPayload In: \n%v", dmSerial)

		payloadLen := calcDMPayloadLen(net)

		// FIXME: Why does this one return an error when the
		// other doesn't!?
		ciphertext, err := dm.Cipher.EncryptSelf(dmSerial, myPrivateKey,
			partnerPubKey, payloadLen)
		if err != nil {
			return
		}

		fpBytes, encryptedPayload, mac, err := createCMIXFields(
			ciphertext, payloadLen, rng)

		fp = format.NewFingerprint(fpBytes)

		return fp, service, encryptedPayload, mac, err
	}
	return net.SendWithAssembler(myID, selfAssemble, params)
}

// makeChaDebugTag is a debug helper that creates non-unique msg identifier.
//
// This is set as the debug tag on messages and enables some level of tracing a
// message (if its contents/chan/type are unique).
func makeDebugTag(id ed25519.PublicKey,
	msg []byte, baseTag string) string {

	h, _ := blake2b.New256(nil)
	h.Write(msg)
	h.Write(id)

	tripCode := base64.RawStdEncoding.EncodeToString(h.Sum(nil))[:12]
	return fmt.Sprintf("%s-%s", baseTag, tripCode)
}

func calcDMPayloadLen(net cMixClient) int {
	// As we don't use the mac or fp fields, we can extend
	// our payload size
	// (-2 to eliminate the first byte of mac and fp)
	return (net.GetMaxMessageLength() +
		format.MacLen + format.KeyFPLen - 2)

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
