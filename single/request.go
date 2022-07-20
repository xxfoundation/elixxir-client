package single

import (
	"bytes"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	cmixMsg "gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/single/message"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/e2e/auth"
	"gitlab.com/elixxir/crypto/e2e/singleUse"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/netTime"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

// Error messages.
const (
	// TransmitRequest
	errPayloadSize     = "size of payload %d exceeds the maximum size of %d (%s for %s)"
	errNetworkHealth   = "cannot send singe-use request when the network is not healthy"
	errMakeDhKeys      = "failed to generate DH keys (%s for %s): %+v"
	errMakeIDs         = "failed to generate IDs (%s for %s): %+v"
	errAddFingerprint  = "failed to add fingerprint %d of %d: %+v (%s for %s)"
	errSendRequest     = "failed to send %s request to %s: %+v"
	errSendRequestPart = "%d requests failed to send, the request will be handleable and will time out"

	// generateDhKeys
	errGenerateInGroup = "failed to generate private key in group: %+v"

	// makeIDs
	errMakeNonce      = "failed to generate nonce: %+v"
	errNewEphemeralID = "failed to generate address ID from newly generated ID: %+v"

	// waitForTimeout
	errResponseTimeout = "waiting for response to single-use request timed out after %s"
)

// Maximum number of request part cMix messages.
const maxNumRequestParts = 255

// GetMaxRequestSize returns the maximum size of a request payload.
func GetMaxRequestSize(net Cmix, e2eGrp *cyclic.Group) int {
	payloadSize := message.GetRequestPayloadSize(
		net.GetMaxMessageLength(), e2eGrp.GetP().ByteLen())
	requestSize := message.GetRequestContentsSize(payloadSize)
	requestPartSize := message.GetRequestPartContentsSize(
		net.GetMaxMessageLength())
	return requestSize + (maxNumRequestParts * requestPartSize)
}

/* Single is a system which allows for an end-to-end encrypted anonymous request
   to be sent to another cMix client, and for them to respond. The system allows
   for communication over the mixnet without an interactive key negotiation
   because the payload inherently carries the negotiation with it. When sending
   a new request, a client create a new discreet log DH keypair as well as a new
   ID. As a result of the fact that the client never identifies itself, the
   system allows the client to stay anonymous while contacting the remote.
*/

// TransmitRequest sends a request to the recipient with the given tag
// containing the given payload. The request is identified as coming from a new
// user ID and the recipient of the request responds to that address. As a
// result, this request does not reveal the identity of the sender.
//
// The current implementation allows for up to maxNumRequestParts cMix request
// payloads. GetMaxRequestSize can be used to get the max size.
//
// The network follower must be running and healthy to transmit.
func TransmitRequest(recipient contact.Contact, tag string, payload []byte,
	responseCB Response, params RequestParams, net Cmix, rng csprng.Source,
	e2eGrp *cyclic.Group) ([]id.Round, receptionID.EphemeralIdentity, error) {

	if len(payload) > GetMaxRequestSize(net, e2eGrp) {
		return nil, receptionID.EphemeralIdentity{}, errors.Errorf(
			errPayloadSize, len(payload), GetMaxRequestSize(net, e2eGrp), tag,
			recipient)
	}

	if !net.IsHealthy() {
		return nil, receptionID.EphemeralIdentity{},
			errors.New(errNetworkHealth)
	}

	// Get address ID address space size; this blocks until the address space
	// size is set for the first time
	addressSize := net.GetAddressSpace()
	timeStart := netTime.Now()

	// Generate DH key and public key
	dhKey, publicKey, err := generateDhKeys(e2eGrp, recipient.DhPubKey, rng)
	if err != nil {
		return nil, receptionID.EphemeralIdentity{},
			errors.Errorf(errMakeDhKeys, tag, recipient, err)
	}

	// Build the message payload
	payloadSize := message.GetRequestPayloadSize(net.GetMaxMessageLength(),
		e2eGrp.GetP().ByteLen())
	firstPart, parts := partitionPayload(
		message.GetRequestContentsSize(payloadSize),
		message.GetRequestPartContentsSize(net.GetMaxMessageLength()),
		payload)
	request := message.NewRequest(
		net.GetMaxMessageLength(), e2eGrp.GetP().ByteLen())
	requestPayload := message.NewRequestPayload(
		request.GetPayloadSize(), firstPart, params.MaxResponseMessages)
	requestPayload.SetNumRequestParts(1 + uint8(len(parts)))

	// Generate new user ID and address ID
	var sendingID receptionID.EphemeralIdentity
	requestPayload, sendingID, err = makeIDs(
		requestPayload, publicKey, addressSize, params.Timeout, timeStart, rng)
	if err != nil {
		return nil, receptionID.EphemeralIdentity{},
			errors.Errorf(errMakeIDs, tag, recipient, err)
	}

	// Encrypt and assemble payload
	fp := singleUse.NewRequestFingerprint(recipient.DhPubKey)
	key := singleUse.NewRequestKey(dhKey)
	encryptedPayload := auth.Crypt(key, fp[:24], requestPayload.Marshal())

	// Generate cMix message MAC
	mac := singleUse.MakeMAC(key, encryptedPayload)

	// Assemble the payload
	request.SetPubKey(publicKey)
	request.SetPayload(encryptedPayload)

	// Register the response pickup
	collator := message.NewCollator(params.MaxResponseMessages)
	timeoutKillChan := make(chan bool)
	var callbackOnce sync.Once
	wrapper := func(payload []byte, receptionID receptionID.EphemeralIdentity,
		rounds []rounds.Round, err error) {
		select {
		case timeoutKillChan <- true:
		default:
		}

		callbackOnce.Do(func() {
			net.DeleteClientFingerprints(sendingID.Source)
			go responseCB.Callback(payload, receptionID, rounds, err)
		})
	}

	cyphers := makeCyphers(dhKey, params.MaxResponseMessages,
		singleUse.NewResponseKey, singleUse.NewResponseFingerprint)

	roundIds := newRoundIdCollector(len(cyphers))
	for i, cy := range cyphers {
		processor := responseProcessor{
			sendingID: sendingID,
			c:         collator,
			callback:  wrapper,
			cy:        cy,
			tag:       tag,
			recipient: &recipient,
			roundIDs:  roundIds,
		}

		err = net.AddFingerprint(
			sendingID.Source, processor.cy.getFingerprint(), &processor)
		if err != nil {
			return nil, receptionID.EphemeralIdentity{}, errors.Errorf(
				errAddFingerprint, i, len(cyphers), tag, recipient, err)
		}
	}

	net.AddIdentity(sendingID.Source, timeStart.Add(params.Timeout), false)

	// Send the payload
	svc := cmixMsg.Service{
		Identifier: recipient.ID[:],
		Tag:        tag,
		Metadata:   nil,
	}
	params.CmixParams.Timeout = params.Timeout
	if params.CmixParams.DebugTag == cmix.DefaultDebugTag ||
		params.CmixParams.DebugTag == "" {
		params.CmixParams.DebugTag = "single-use.Request"
	}

	jww.INFO.Printf("[SU] Sending single-use request cMix message with %d "+
		"parts to %s (%s).", 1+len(parts), recipient.ID, tag)

	rid, ephID, err := net.Send(
		recipient.ID, fp, svc, request.Marshal(), mac, params.CmixParams)
	if err != nil {
		return nil, receptionID.EphemeralIdentity{},
			errors.Errorf(errSendRequest, tag, recipient, err)
	}

	jww.DEBUG.Printf("[SU] Sent single-use request cMix message part "+
		"%d of %d on round %d to %s (eph ID %d) (%s).",
		0, len(parts)+1, rid, recipient.ID, ephID.Int64(), tag)

	var wg sync.WaitGroup
	wg.Add(len(parts))
	failed := uint32(0)

	roundIDs := make([]id.Round, len(parts)+1)
	roundIDs[0] = rid
	for i, part := range parts {
		go func(i int, part []byte) {
			defer wg.Done()
			requestPart := message.NewRequestPart(net.GetMaxMessageLength())
			requestPart.SetPartNum(uint8(i))
			requestPart.SetContents(part)

			key := singleUse.NewRequestPartKey(dhKey, uint64(i))
			fp := singleUse.NewRequestPartFingerprint(dhKey, uint64(i))
			encryptedPayload := auth.Crypt(key, fp[:24], requestPart.Marshal())
			mac := singleUse.MakeMAC(key, encryptedPayload)

			var ephID ephemeral.Id
			var err error
			roundIDs[i], ephID, err = net.Send(recipient.ID, fp,
				cmixMsg.Service{}, encryptedPayload, mac, params.CmixParams)
			if err != nil {
				atomic.AddUint32(&failed, 1)
				jww.ERROR.Printf("[SU] Failed to send single-use request "+
					"cMix message part %d of %d to %s (%s): %+v",
					i, len(part)+1, recipient.ID, tag, err)
				return
			}

			jww.DEBUG.Printf("[SU] Sent single-use request cMix message part "+
				"%d of %d on round %d to %s (eph ID %d) (%s).", i,
				len(parts)+1, roundIDs[i], recipient.ID, ephID.Int64(), tag)
		}(i+1, part)
	}

	// Wait for all go routines to finish
	wg.Wait()

	if failed > 0 {
		return nil, receptionID.EphemeralIdentity{},
			errors.Errorf(errSendRequestPart, failed)
	}

	jww.INFO.Printf("[SU] Sent single-use request cMix message with %d "+
		"parts to %s (%s).", 1+len(parts), recipient.ID, tag)

	remainingTimeout := params.Timeout - netTime.Since(timeStart)
	go waitForTimeout(timeoutKillChan, wrapper, remainingTimeout)

	return []id.Round{rid}, sendingID, nil
}

// generateDhKeys generates a new public key and DH key.
func generateDhKeys(grp *cyclic.Group, dhPubKey *cyclic.Int, rng io.Reader) (
	dhKey, publicKey *cyclic.Int, err error) {

	// Generate private key
	privKeyBytes, err := csprng.GenerateInGroup(
		grp.GetP().Bytes(), grp.GetP().ByteLen(), rng)
	if err != nil {
		return nil, nil, errors.Errorf(errGenerateInGroup, err)
	}
	privKey := grp.NewIntFromBytes(privKeyBytes)

	// Generate public key and DH key
	publicKey = diffieHellman.GeneratePublicKey(privKey, grp)
	dhKey = grp.Exp(dhPubKey, privKey, grp.NewInt(1))

	return dhKey, publicKey, nil
}

// makeIDs generates a new user ID and address ID with a start and end within
// the given timout. The ID is generated from the unencrypted msg payload, which
// contains a nonce. If the generated address ID has a window that is not within
// +/- the given 2*Timeout from now, then the IDs are generated again using a
// new nonce.
func makeIDs(payload message.RequestPayload, publicKey *cyclic.Int,
	addressSize uint8, timeout time.Duration, timeNow time.Time, rng io.Reader) (
	message.RequestPayload, receptionID.EphemeralIdentity, error) {
	var rid *id.ID
	var ephID ephemeral.Id

	// Generate acceptable window for the address ID to exist in
	windowStart, windowEnd := timeNow.Add(-2*timeout), timeNow.Add(2*timeout)
	start, end := timeNow, timeNow

	// Loop until the address ID's start and end are within bounds
	for windowStart.Before(start) || windowEnd.After(end) {
		// Generate new nonce
		err := payload.SetNonce(rng)
		if err != nil {
			return message.RequestPayload{}, receptionID.EphemeralIdentity{},
				errors.Errorf(errMakeNonce, err)
		}

		// Generate ID from unencrypted payload
		rid = payload.GetRecipientID(publicKey)

		// Generate the address ID
		ephID, start, end, err = ephemeral.GetId(
			rid, uint(addressSize), timeNow.UnixNano())
		if err != nil {
			return message.RequestPayload{}, receptionID.EphemeralIdentity{},
				errors.Errorf(errNewEphemeralID, err)
		}
	}

	jww.INFO.Printf("[SU] Generated singe-use sender reception ID: %s, "+
		"ephId: %d, publicKey: %s, payload: %q",
		rid, ephID.Int64(), publicKey.Text(10), payload)

	return payload, receptionID.EphemeralIdentity{
		EphId:  ephID,
		Source: rid,
	}, nil
}

// waitForTimeout is a long-running thread which handles timing out a request.
// It can be canceled by channel.
func waitForTimeout(kill chan bool, cb callbackWrapper, timeout time.Duration) {
	select {
	case <-kill:
		return
	case <-time.After(timeout):
		cb(nil, receptionID.EphemeralIdentity{}, nil,
			errors.Errorf(errResponseTimeout, timeout))
	}
}

// partitionPayload splits the payload into its parts. The first part is of size
// firstPartSize and is shorter than the rest since it is sent in the
// message.Request, which includes extra information. It is also returned on its
// own so that it can be handled on its own. The rest of the parts are of size
// partSize and will be sent as part of message.RequestPart.
func partitionPayload(firstPartSize, partSize int, payload []byte) (
	firstPart []byte, parts [][]byte) {

	// Return just the first part if it fits in a single message
	if len(payload) <= firstPartSize {
		return payload, nil
	}

	firstPart = payload[:firstPartSize]

	numParts := (len(payload[:firstPartSize]) + partSize - 1) / partSize
	parts = make([][]byte, 0, numParts)
	buff := bytes.NewBuffer(payload[firstPartSize:])

	for n := buff.Next(partSize); len(n) > 0; n = buff.Next(partSize) {
		newPart := make([]byte, partSize)
		copy(newPart, n)
		parts = append(parts, newPart)
	}

	return firstPart, parts
}
