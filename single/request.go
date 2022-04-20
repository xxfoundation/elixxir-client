package single

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	cmixMsg "gitlab.com/elixxir/client/cmix/message"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/single/message"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/e2e/auth"
	"gitlab.com/elixxir/crypto/e2e/singleUse"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/netTime"
	"io"
	"sync"
	"time"
)

// Response interface allows for callbacks to
type Response interface {
	Callback(payload []byte, receptionID receptionID.EphemeralIdentity,
		round rounds.Round, err error)
}

type RequestParams struct {
	Timeout             time.Duration
	MaxResponseMessages uint8
	CmixParam           cmix.CMIXParams
}

func GetDefaultRequestParams() RequestParams {
	return RequestParams{
		Timeout:             30 * time.Second,
		MaxResponseMessages: 255,
		CmixParam:           cmix.GetDefaultCMIXParams(),
	}
}

// Error messages.
const (
	// TransmitRequest
	errNetworkHealth  = "cannot send singe-use request when the network is not healthy"
	errMakeDhKeys     = "failed to generate DH keys (%s for %s): %+v"
	errMakeIDs        = "failed to generate IDs (%s for %s): %+v"
	errAddFingerprint = "failed to add fingerprint %d of %d: %+v (%s for %s)"
	errSendRequest    = "failed to send %s request to %s: %+v"

	// generateDhKeys
	errGenerateInGroup = "failed to generate private key in group: %+v"

	// makeIDs
	errMakeNonce      = "failed to generate nonce: %+v"
	errNewEphemeralID = "failed to generate address ID from newly generated ID: %+v"

	// waitForTimeout
	errResponseTimeout = "waiting for response to single-use request timed out after %s"
)

// GetMaxRequestSize returns the maximum size of a request payload.
func GetMaxRequestSize(net cmix.Client, e2eGrp *cyclic.Group) int {
	payloadSize := message.GetRequestPayloadSize(net.GetMaxMessageLength(),
		e2eGrp.GetP().ByteLen())
	return message.GetRequestContentsSize(payloadSize)
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
// The current implementation only allows for a single cMix request payload.
// Because the request payload itself must include negotiation materials, it is
// limited to just a few thousand bits of payload, and will return an error if
// the payload is too large. GetMaxRequestSize can be used to get this max size.
// The network follower must be running and healthy to transmit.
func TransmitRequest(recipient contact.Contact, tag string, payload []byte,
	callback Response, param RequestParams, net cmix.Client, rng csprng.Source,
	e2eGrp *cyclic.Group) (id.Round, receptionID.EphemeralIdentity, error) {
	if !net.IsHealthy() {
		return 0, receptionID.EphemeralIdentity{}, errors.New(errNetworkHealth)
	}

	// Get address ID address space size; this blocks until the address space
	// size is set for the first time
	addressSize := net.GetAddressSpace()
	timeStart := netTime.Now()

	// Generate DH key and public key
	dhKey, publicKey, err := generateDhKeys(e2eGrp, recipient.DhPubKey, rng)
	if err != nil {
		return 0, receptionID.EphemeralIdentity{},
			errors.Errorf(errMakeDhKeys, tag, recipient, err)
	}

	// Build the message payload
	request := message.NewRequest(
		net.GetMaxMessageLength(), e2eGrp.GetP().ByteLen())
	requestPayload := message.NewRequestPayload(
		request.GetPayloadSize(), payload, param.MaxResponseMessages)

	// Generate new user ID and address ID
	var sendingID receptionID.EphemeralIdentity
	requestPayload, sendingID, err = makeIDs(
		requestPayload, publicKey, addressSize, param.Timeout, timeStart, rng)
	if err != nil {
		return 0, receptionID.EphemeralIdentity{},
			errors.Errorf(errMakeIDs, tag, recipient, err)
	}

	// Encrypt and assemble payload
	fp := singleUse.NewTransmitFingerprint(recipient.DhPubKey)
	key := singleUse.NewTransmitKey(dhKey)
	encryptedPayload := auth.Crypt(key, fp[:24], requestPayload.Marshal())
	request.SetPubKey(publicKey)
	request.SetPayload(encryptedPayload)

	// Generate cMix message MAC
	mac := singleUse.MakeMAC(key, encryptedPayload)

	// Register the response pickup
	collator := message.NewCollator(param.MaxResponseMessages)
	timeoutKillChan := make(chan bool)
	var callbackOnce sync.Once
	wrapper := func(payload []byte, receptionID receptionID.EphemeralIdentity,
		round rounds.Round, err error) {
		select {
		case timeoutKillChan <- true:
		default:
		}

		callbackOnce.Do(func() {
			net.DeleteClientFingerprints(sendingID.Source)
			go callback.Callback(payload, receptionID, round, err)
		})
	}

	cyphers := makeCyphers(dhKey, param.MaxResponseMessages)

	for i, cy := range cyphers {
		processor := responseProcessor{
			sendingID: sendingID,
			c:         collator,
			callback:  wrapper,
			cy:        cy,
			tag:       tag,
			recipient: &recipient,
		}

		err = net.AddFingerprint(
			sendingID.Source, processor.cy.GetFingerprint(), &processor)
		if err != nil {
			return 0, receptionID.EphemeralIdentity{}, errors.Errorf(
				errAddFingerprint, i, len(cyphers), tag, recipient, err)
		}
	}

	net.AddIdentity(sendingID.Source, timeStart.Add(param.Timeout), false)

	// Send the payload
	svc := cmixMsg.Service{
		Identifier: recipient.ID[:],
		Tag:        tag,
		Metadata:   nil,
	}
	param.CmixParam.Timeout = param.Timeout

	rid, _, err := net.Send(recipient.ID, cmixMsg.RandomFingerprint(rng), svc,
		request.Marshal(), mac, param.CmixParam)
	if err != nil {
		return 0, receptionID.EphemeralIdentity{},
			errors.Errorf(errSendRequest, tag, recipient, err)
	}

	remainingTimeout := param.Timeout - netTime.Since(timeStart)
	go waitForTimeout(timeoutKillChan, wrapper, remainingTimeout)

	return rid, sendingID, nil
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
	publicKey = grp.ExpG(privKey, grp.NewInt(1))
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
		"ephId: %d, publicKey: %x, payload: %q",
		rid, ephID.Int64(), publicKey.Bytes(), payload)

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
		cb(
			nil,
			receptionID.EphemeralIdentity{},
			rounds.Round{},
			errors.Errorf(errResponseTimeout, timeout),
		)
	}
}