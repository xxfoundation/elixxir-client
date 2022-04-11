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

type Response interface {
	Callback(payload []byte, receptionID receptionID.EphemeralIdentity,
		round rounds.Round, err error)
}

type RequestParams struct {
	Timeout     time.Duration
	MaxMessages uint8
	CmixParam   cmix.CMIXParams
}

func TransmitRequest(recipient contact.Contact, tag string, payload []byte,
	callback Response, param RequestParams, net cmix.Client, rng csprng.Source,
	e2eGrp *cyclic.Group) (id.Round, receptionID.EphemeralIdentity, error) {
	// get address ID address space size; this blocks until the address space
	// size is set for the first time
	addressSize := net.GetAddressSpace()
	timeStart := netTime.Now()

	// Generate DH key and public key
	dhKey, publicKey, err := generateDhKeys(e2eGrp, recipient.DhPubKey, rng)
	if err != nil {
		return 0, receptionID.EphemeralIdentity{}, err
	}

	//build the message payload
	request := message.NewRequest(net.GetMaxMessageLength(),
		e2eGrp.GetP().ByteLen())
	requestPayload := message.NewRequestPayload(request.GetPayloadSize(),
		payload, param.MaxMessages)

	// Generate new user ID and address ID
	var sendingID receptionID.EphemeralIdentity
	requestPayload, sendingID, err = makeIDs(requestPayload, publicKey, addressSize, param.Timeout,
		timeStart, rng)
	if err != nil {
		return 0, receptionID.EphemeralIdentity{}, errors.Errorf("failed to generate IDs: %+v", err)
	}

	// Encrypt payload
	fp := singleUse.NewTransmitFingerprint(recipient.DhPubKey)
	key := singleUse.NewTransmitKey(dhKey)
	encryptedPayload := auth.Crypt(key, fp[:24], requestPayload.Marshal())

	// Generate CMIX message MAC
	mac := singleUse.MakeMAC(key, encryptedPayload)

	//assemble the payload
	request.SetPubKey(publicKey)
	request.SetPayload(encryptedPayload)

	//register the response pickup
	collator := message.NewCollator(param.MaxMessages)
	timeoutKillChan := make(chan bool)
	callbackOnce := sync.Once{}
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

	cyphers := makeCyphers(dhKey, param.MaxMessages)

	for i := uint8(0); i < param.MaxMessages; i++ {
		processor := responceProcessor{
			sendingID: sendingID,
			c:         collator,
			callback:  wrapper,
			cy:        cyphers[i],
			tag:       tag,
			recipient: &recipient,
		}

		if err = net.AddFingerprint(sendingID.Source, processor.cy.GetFingerprint(),
			&processor); err != nil {
			return 0, receptionID.EphemeralIdentity{},
				errors.Errorf("failed to add fingerprint %d IDs: %+v", i, err)
		}
	}

	net.AddIdentity(sendingID.Source, timeStart.Add(param.Timeout), false)

	//send the payload
	svc := cmixMsg.Service{
		Identifier: recipient.ID[:],
		Tag:        tag,
		Metadata:   nil,
	}
	param.CmixParam.Timeout = param.Timeout

	rid, _, err := net.Send(recipient.ID, cmixMsg.RandomFingerprint(rng), svc, request.Marshal(), mac,
		param.CmixParam)

	if err != nil {
		return 0, receptionID.EphemeralIdentity{}, err
	}

	remainingTimeout := param.Timeout - netTime.Since(timeStart)
	go waitForTimeout(timeoutKillChan, wrapper, remainingTimeout)

	return rid, sendingID, nil
}

// generateDhKeys generates a new public key and DH key.
func generateDhKeys(grp *cyclic.Group, dhPubKey *cyclic.Int,
	rng io.Reader) (*cyclic.Int, *cyclic.Int, error) {
	// Generate private key
	privKeyBytes, err := csprng.GenerateInGroup(grp.GetP().Bytes(),
		grp.GetP().ByteLen(), rng)
	if err != nil {
		return nil, nil, errors.Errorf("failed to generate key in group: %+v",
			err)
	}
	privKey := grp.NewIntFromBytes(privKeyBytes)

	// Generate public key and DH key
	publicKey := grp.ExpG(privKey, grp.NewInt(1))
	dhKey := grp.Exp(dhPubKey, privKey, grp.NewInt(1))
	return dhKey, publicKey, nil
}

// makeIDs generates a new user ID and address ID with a start and end within
// the given timout. The ID is generated from the unencrypted msg payload, which
// contains a nonce. If the generated address ID has a window that is not
// within +/- the given 2*Timeout from now, then the IDs are generated again
// using a new nonce.
func makeIDs(msg message.RequestPayload, publicKey *cyclic.Int,
	addressSize uint8, timeout time.Duration, timeNow time.Time,
	rng io.Reader) (message.RequestPayload, receptionID.EphemeralIdentity, error) {
	var rid *id.ID
	var ephID ephemeral.Id

	// Generate acceptable window for the address ID to exist in
	windowStart, windowEnd := timeNow.Add(-2*timeout), timeNow.Add(2*timeout)
	start, end := timeNow, timeNow

	// Loop until the address ID's start and end are within bounds
	for windowStart.Before(start) || windowEnd.After(end) {
		// Generate new nonce
		err := msg.SetNonce(rng)
		if err != nil {
			return message.RequestPayload{}, receptionID.EphemeralIdentity{},
				errors.Errorf("failed to generate nonce: %+v", err)
		}

		// Generate ID from unencrypted payload
		rid = msg.GetRID(publicKey)

		// Generate the address ID
		ephID, start, end, err = ephemeral.GetId(rid, uint(addressSize),
			timeNow.UnixNano())
		if err != nil {
			return message.RequestPayload{}, receptionID.EphemeralIdentity{},
				errors.Errorf("failed to generate "+
					"address ID from newly generated ID: %+v", err)
		}
		jww.DEBUG.Printf("address.GetId(%s, %d, %d) = %d", rid,
			addressSize, timeNow.UnixNano(), ephID.Int64())
	}

	jww.INFO.Printf("generated by singe use sender reception id for single use: %s, "+
		"ephId: %d, pubkey: %x, msg: %s", rid, ephID.Int64(), publicKey.Bytes(), msg)

	return msg, receptionID.EphemeralIdentity{
		EphId:  ephID,
		Source: rid,
	}, nil
}

func waitForTimeout(kill chan bool, callback callbackWrapper, timeout time.Duration) {
	timer := time.NewTimer(timeout)
	select {
	case <-kill:
		return
	case <-timer.C:
		err := errors.Errorf("waiting for response to single-use transmission "+
			"timed out after %s.", timeout)
		callback(nil, receptionID.EphemeralIdentity{}, rounds.Round{}, err)
	}
}
