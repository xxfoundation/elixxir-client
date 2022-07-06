package bindings

import (
	"encoding/json"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/single"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/xx_network/primitives/id"
)

/* PUBLIC WRAPPER METHODS */

// TransmitSingleUse accepts a marshalled recipient contact object, tag, payload, SingleUseResponse callback func & a
// Client.  Transmits payload to recipient via single use
func TransmitSingleUse(e2eID int, recipient []byte, tag string, payload []byte, responseCB SingleUseResponse) ([]byte, error) {
	e2eCl, err := e2eTrackerSingleton.get(e2eID)
	if err != nil {
		return nil, err
	}

	recipientContact, err := contact.Unmarshal(recipient)
	if err != nil {
		return nil, err
	}

	rcb := &singleUseResponse{response: responseCB}

	rids, eid, err := single.TransmitRequest(recipientContact, tag, payload, rcb, single.GetDefaultRequestParams(), e2eCl.api.GetCmix(), e2eCl.api.GetRng().GetStream(), e2eCl.api.GetStorage().GetE2EGroup())

	if err != nil {
		return nil, err
	}
	sr := SingleUseSendReport{
		EphID:       eid.EphId.Int64(),
		ReceptionID: eid.Source.Marshal(),
		RoundsList:  makeRoundsList(rids),
	}
	return json.Marshal(sr)
}

// Listen starts a single use listener on a given tag using the passed in client and SingleUseCallback func
func Listen(e2eID int, tag string, cb SingleUseCallback) (StopFunc, error) {
	e2eCl, err := e2eTrackerSingleton.get(e2eID)
	if err != nil {
		return nil, err
	}

	listener := singleUseListener{scb: cb}
	dhpk, err := e2eCl.api.GetReceptionIdentity().GetDHKeyPrivate()
	if err != nil {
		return nil, err
	}
	l := single.Listen(tag, e2eCl.api.GetReceptionIdentity().ID, dhpk, e2eCl.api.GetCmix(), e2eCl.api.GetStorage().GetE2EGroup(), listener)
	return l.Stop, nil
}

// JSON Types

// SingleUseSendReport is the bindings struct used to represent information returned by single.TransmitRequest
//
// Example json marshalled struct:
// {"Rounds":[1,5,9],
//  "EphID":{"EphId":[0,0,0,0,0,0,3,89],
//  "Source":"emV6aW1hAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD"}}
type SingleUseSendReport struct {
	RoundsList
	ReceptionID []byte
	EphID       int64
}

// SingleUseResponseReport is the bindings struct used to represent information passed
// to the single.Response callback interface in response to single.TransmitRequest
//
// Example json marshalled struct:
// {"Rounds":[1,5,9],
//  "Payload":"rSuPD35ELWwm5KTR9ViKIz/r1YGRgXIl5792SF8o8piZzN6sT4Liq4rUU/nfOPvQEjbfWNh/NYxdJ72VctDnWw==",
//  "ReceptionID":{"EphId":[0,0,0,0,0,0,3,89],
//  "Source":"emV6aW1hAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD"},
//  "Err":null}
type SingleUseResponseReport struct {
	RoundsList
	Payload     []byte
	ReceptionID []byte
	EphID       int64
	Err         error
}

// SingleUseCallbackReport is the bindings struct used to represent single use messages
// received by a callback passed into single.Listen
//
// Example json marshalled struct:
// {"Rounds":[1,5,9],
//  "Payload":"rSuPD35ELWwm5KTR9ViKIz/r1YGRgXIl5792SF8o8piZzN6sT4Liq4rUU/nfOPvQEjbfWNh/NYxdJ72VctDnWw==",
//  "Partner":"emV6aW1hAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD",
//  "EphID":{"EphId":[0,0,0,0,0,0,3,89],
//  "Source":"emV6aW1hAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD"}}
type SingleUseCallbackReport struct {
	RoundsList
	Payload     []byte
	Partner     *id.ID
	EphID       int64
	ReceptionID []byte
}

// Function types

// StopFunc is the function to stop a listener returned to the bindings layer when one is started
type StopFunc func()

// SingleUseCallback func is passed into Listen and called when messages are received
// Accepts a SingleUseCallbackReport marshalled to json
type SingleUseCallback interface {
	Callback(callbackReport []byte, err error)
}

// SingleUseResponse is the public facing callback func passed by bindings clients into TransmitSingleUse
// Accepts a SingleUseResponseReport marshalled to json
type SingleUseResponse interface {
	Callback(responseReport []byte, err error)
}

/* CALLBACK WRAPPERS */

/* listener struct */

// singleUseListener is the internal struct used to wrap a SingleUseCallback func,
// which matches the single.Receiver interface
type singleUseListener struct {
	scb SingleUseCallback
}

// Callback is called whenever a single use message is heard by the listener, and translates the info to
//a SingleUseCallbackReport which is marshalled & passed to bindings
func (sl singleUseListener) Callback(req *single.Request, eid receptionID.EphemeralIdentity, rl []rounds.Round) {
	var rids []id.Round
	for _, r := range rl {
		rids = append(rids, r.ID)
	}

	// Todo: what other info from req needs to get to bindings
	scr := SingleUseCallbackReport{
		Payload:     req.GetPayload(),
		RoundsList:  makeRoundsList(rids),
		Partner:     req.GetPartner(),
		EphID:       eid.EphId.Int64(),
		ReceptionID: eid.Source.Marshal(),
	}

	sl.scb.Callback(json.Marshal(scr))
}

/* response struct */

// singleUseResponse is the private struct backing SingleUseResponse, which subscribes to the single.Response interface
type singleUseResponse struct {
	response SingleUseResponse
}

// Callback builds a SingleUseSendReport & passes the json marshalled version into the callback
func (sr singleUseResponse) Callback(payload []byte, receptionID receptionID.EphemeralIdentity,
	rounds []rounds.Round, err error) {
	var rids []id.Round
	for _, r := range rounds {
		rids = append(rids, r.ID)
	}
	sendReport := SingleUseResponseReport{
		RoundsList:  makeRoundsList(rids),
		ReceptionID: receptionID.Source.Marshal(),
		EphID:       receptionID.EphId.Int64(),
		Payload:     payload,
		Err:         err,
	}
	sr.response.Callback(json.Marshal(&sendReport))
}
