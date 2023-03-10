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
	"gitlab.com/elixxir/crypto/fastRNG"
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

	directMessageDebugTag = "dm"
	// The size of the nonce used in the message ID.
	messageNonceSize = 4
)

// SendText is used to send a formatted message to another user.
func (dc *dmClient) SendText(partnerPubKey *ed25519.PublicKey,
	partnerToken uint32,
	msg string, params cmix.CMIXParams) (
	cryptoMessage.ID, rounds.Round, ephemeral.Id, error) {

	pubKeyStr := base64.RawStdEncoding.EncodeToString(*partnerPubKey)

	tag := makeDebugTag(*partnerPubKey, []byte(msg), SendReplyTag)
	jww.INFO.Printf("[DM][%s] SendText(%s)", tag, pubKeyStr)
	txt := &Text{
		Version: textVersion,
		Text:    msg,
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

// SendDMReply is used to send a formatted direct message reply.
//
// If the message ID that the reply is sent to does not exist,
// then the other side will post the message as a normal
// message and not as a reply.
func (dc *dmClient) SendReply(partnerPubKey *ed25519.PublicKey,
	partnerToken uint32, msg string, replyTo cryptoMessage.ID,
	params cmix.CMIXParams) (cryptoMessage.ID, rounds.Round,
	ephemeral.Id, error) {

	pubKeyStr := base64.RawStdEncoding.EncodeToString(*partnerPubKey)

	tag := makeDebugTag(*partnerPubKey, []byte(msg), SendReplyTag)
	jww.INFO.Printf("[DM][%s] SendReply(%s, to %s)", tag, pubKeyStr,
		replyTo)
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

	return dc.Send(partnerPubKey, partnerToken, ReplyType, txtMarshaled,
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
	jww.INFO.Printf("[DM][%s] SendReaction(%s, to %s)", tag,
		base64.RawStdEncoding.EncodeToString(*partnerPubKey),
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

	partnerID := deriveReceptionID(partnerPubKey.Bytes(), partnerToken)

	// Note: We log sends on exit, and append what happened to the message
	// this cuts down on clutter in the log.
	sendPrint := fmt.Sprintf("[DM][%s] Sending dm to %s type %d at %s",
		params.DebugTag, partnerID, messageType, netTime.Now())
	defer func() { jww.INFO.Println(sendPrint) }()

	rng := dc.rng.GetStream()
	defer rng.Close()

	nickname, _ := dc.nm.GetNickname()

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

	if params.DebugTag == cmix.DefaultDebugTag {
		params.DebugTag = directMessageDebugTag
	}

	sendPrint += fmt.Sprintf(", pending send %s", netTime.Now())
	uuid, err := dc.st.DenotePendingSend(*partnerEdwardsPubKey,
		dc.me.PubKey, partnerToken, messageType, directMessage)
	if err != nil {
		sendPrint += fmt.Sprintf(", pending send failed %s",
			err.Error())
		errDenote := dc.st.FailedSend(uuid)
		if errDenote != nil {
			sendPrint += fmt.Sprintf(
				", failed to denote failed dm send: %s",
				errDenote.Error())
		}
		return cryptoMessage.ID{}, rounds.Round{},
			ephemeral.Id{}, err
	}

	rndID, ephIDs, err := send(dc.net, dc.selfReceptionID,
		partnerID, partnerPubKey, dc.privateKey, partnerToken,
		directMessage, params, dc.rng)
	if err != nil {
		sendPrint += fmt.Sprintf(", err on send: %+v", err)
		errDenote := dc.st.FailedSend(uuid)
		if errDenote != nil {
			sendPrint += fmt.Sprintf(
				", failed to denote failed dm send: %s",
				errDenote.Error())
		}
		return cryptoMessage.ID{}, rounds.Round{},
			ephemeral.Id{}, err
	}

	// Now that we have a round ID, derive the msgID
	jww.INFO.Printf("[DM] DeriveDirectMessage(%s...) Send", partnerID)
	msgID := cryptoMessage.DeriveDirectMessageID(partnerID,
		directMessage)

	sendPrint += fmt.Sprintf(", send eph %v rnd %s MsgID %s",
		ephIDs, rndID.ID, msgID)

	err = dc.st.Sent(uuid, msgID, rndID)
	if err != nil {
		sendPrint += fmt.Sprintf(", dm send denote failed: %s ",
			err.Error())
	}
	return msgID, rndID, ephIDs[1], err

}

// DeriveReceptionID returns a reception ID for direct messages sent
// to the user. It generates this ID by hashing the public key and
// an arbitrary idToken together. The ID type is set to "User".
func DeriveReceptionID(publicKey ed25519.PublicKey, idToken uint32) *id.ID {
	nikePubKey := ecdh.Edwards2ECDHNIKEPublicKey(&publicKey)
	return deriveReceptionID(nikePubKey.Bytes(), idToken)
}

func deriveReceptionID(keyBytes []byte, idToken uint32) *id.ID {
	h, err := blake2b.New256(nil)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	h.Write(keyBytes)
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

func send(net cMixClient, myID *id.ID, partnerID *id.ID,
	partnerPubKey nike.PublicKey,
	myPrivateKey nike.PrivateKey, partnerToken uint32,
	msg *DirectMessage, params cmix.CMIXParams,
	rngGenerator *fastRNG.StreamGenerator) (rounds.Round,
	[]ephemeral.Id, error) {

	// Send to Partner
	assemble := func(rid id.Round) ([]cmix.TargetedCmixMessage, error) {
		rng := rngGenerator.GetStream()
		defer rng.Close()

		// SEND
		msg.RoundID = uint64(rid)

		// Serialize the message
		dmSerial, err := proto.Marshal(msg)
		if err != nil {
			return nil, err
		}

		service := createRandomService(rng)

		payloadLen := calcDMPayloadLen(net)

		ciphertext := dm.Cipher.Encrypt(dmSerial, myPrivateKey,
			partnerPubKey, rng, payloadLen)

		fpBytes, encryptedPayload, mac, err := createCMIXFields(
			ciphertext, payloadLen, rng)
		if err != nil {
			return nil, err
		}

		fp := format.NewFingerprint(fpBytes)

		sendMsg := cmix.TargetedCmixMessage{
			Recipient:   partnerID,
			Payload:     encryptedPayload,
			Fingerprint: fp,
			Service:     service,
			Mac:         mac,
		}

		// SELF SEND
		// NOTE: We do not modify the round id already in the
		//       message object. This enables the same msgID
		//       on sender and recipient.
		msg.SelfRoundID = uint64(rid)
		// NOTE: Very important to overwrite these fields
		// for self sending!
		msg.DMToken = partnerToken

		// Serialize the message
		selfDMSerial, err := proto.Marshal(msg)
		if err != nil {
			return nil, err
		}

		service = createRandomService(rng)

		payloadLen = calcDMPayloadLen(net)

		// FIXME: Why does this one return an error when the
		// other doesn't!?
		selfCiphertext, err := dm.Cipher.EncryptSelf(selfDMSerial,
			myPrivateKey, partnerPubKey, payloadLen)
		if err != nil {
			return nil, err
		}

		fpBytes, encryptedPayload, mac, err = createCMIXFields(
			selfCiphertext, payloadLen, rng)
		if err != nil {
			return nil, err
		}

		fp = format.NewFingerprint(fpBytes)

		selfSendMsg := cmix.TargetedCmixMessage{
			Recipient:   myID,
			Payload:     encryptedPayload,
			Fingerprint: fp,
			Service:     service,
			Mac:         mac,
		}

		return []cmix.TargetedCmixMessage{sendMsg, selfSendMsg}, nil
	}
	return net.SendManyWithAssembler([]*id.ID{partnerID, myID}, assemble, params)
}

// makeDebugTag is a debug helper that creates non-unique msg identifier.
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
