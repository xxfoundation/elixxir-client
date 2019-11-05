////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Package bot functions for working with the user discovery bot (UDB)
package bots

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/cmixproto"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/primitives/id"
	"strings"
	"time"
)

var pushkeyExpected = "PUSHKEY COMPLETE"
var pushkeyErrorExpected = "Could not push key"

// Register sends a registration message to the UDB. It does this by sending 2
// PUSHKEY messages to the UDB, then calling UDB's REGISTER command.
// If any of the commands fail, it returns an error.
// valueType: Currently only "EMAIL"
func Register(valueType, value string, publicKey []byte, regStatus func(int), timeout time.Duration) error {
	globals.Log.DEBUG.Printf("Running register for %v, %v, %q", valueType,
		value, publicKey)

	registerTimeout := time.NewTimer(timeout)

	var err error
	if valueType == "EMAIL" {
		value, err = hashAndEncode(strings.ToLower(value))
		if err != nil {
			return fmt.Errorf("Could not hash and encode email %s: %+v", value, err)
		}
	}

	keyFP := fingerprint(publicKey)

	regStatus(globals.UDB_REG_PUSHKEY)

	// push key and error if it already exists
	err = pushKey(UdbID, keyFP, publicKey)

	if err != nil {
		return errors.Wrap(err, "Could not PUSHKEY")
	}

	var response string

	// wait for the response to submitting the key against the timeout.
	// discard responses from other searches
	submitted := false

	for !submitted {
		select {
		case response = <-pushKeyResponseListener:
			if strings.Contains(response, keyFP) {
				if strings.Contains(response, pushkeyExpected) {
					submitted = true
				} else {
					err := errors.New(response)
					return errors.Wrap(err, "PushKey failed")
				}
			}
		case <-registerTimeout.C:
			return errors.New("UDB register timeout exceeded on key submission")
		}
	}

	//send the user information to udb
	msgBody := parse.Pack(&parse.TypedBody{
		MessageType: int32(cmixproto.Type_UDB_REGISTER),
		Body:        []byte(fmt.Sprintf("%s %s %s", valueType, value, keyFP)),
	})

	regStatus(globals.UDB_REG_PUSHUSER)

	// Send register command
	// Send register command
	err = sendCommand(UdbID, msgBody)
	if err != nil {
		return errors.Wrap(err, "Could not Push User")
	}

	// wait for the response to submitting the key against the timeout.
	// discard responses from other searches
	complete := false

	for !complete {
		select {
		case response = <-registerResponseListener:
			expected := "REGISTRATION COMPLETE"
			unavalibleReg := "Can not register with existing email"
			if strings.Contains(response, expected) {
				complete = true
			} else if strings.Contains(response, value) && strings.Contains(response, unavalibleReg) {
				return errors.New("Cannot register with existing username")
			}
		case <-registerTimeout.C:
			return errors.New("UDB register timeout exceeded on user submission")
		}
	}

	return nil
}

// Search returns a userID and public key based on the search criteria
// it accepts a valueType of EMAIL and value of an e-mail address, and
// returns a map of userid -> public key
func Search(valueType, value string, searchStatus func(int), timeout time.Duration) (*id.User, []byte, error) {
	globals.Log.DEBUG.Printf("Running search for %v, %v", valueType, value)

	searchTimeout := time.NewTimer(timeout)

	var err error
	if valueType == "EMAIL" {
		value, err = hashAndEncode(strings.ToLower(value))
		if err != nil {
			return nil, nil, fmt.Errorf("Could not hash and encode email %s: %+v", value, err)
		}
	}

	searchStatus(globals.UDB_SEARCH_LOOK)

	msgBody := parse.Pack(&parse.TypedBody{
		MessageType: int32(cmixproto.Type_UDB_SEARCH),
		Body:        []byte(fmt.Sprintf("%s %s", valueType, value)),
	})
	err = sendCommand(UdbID, msgBody)
	if err != nil {
		return nil, nil, err
	}

	var response string

	// wait for the response to searching for the value against the timeout.
	// discard responses from other searches
	found := false

	for !found {
		select {
		case response = <-searchResponseListener:
			empty := fmt.Sprintf("SEARCH %s NOTFOUND", value)
			if response == empty {
				return nil, nil, nil
			}
			if strings.Contains(response, value) {
				found = true
			}
		case <-searchTimeout.C:
			return nil, nil, errors.New("UDB search timeout exceeded on user lookup")
		}
	}

	// While search returns more than 1 result, we only process the first
	cMixUID, keyFP := parseSearch(response)
	if *cMixUID == *id.ZeroID {
		return nil, nil, fmt.Errorf("%s", keyFP)
	}

	searchStatus(globals.UDB_SEARCH_GETKEY)

	// Get the full key and decode it
	msgBody = parse.Pack(&parse.TypedBody{
		MessageType: int32(cmixproto.Type_UDB_GET_KEY),
		Body:        []byte(keyFP),
	})
	err = sendCommand(UdbID, msgBody)
	if err != nil {
		return nil, nil, err
	}

	// wait for the response to searching for the key against the timeout.
	// discard responses from other searches
	found = false
	for !found {
		select {
		case response = <-getKeyResponseListener:
			if strings.Contains(response, keyFP) {
				found = true
			}
		case <-searchTimeout.C:
			return nil, nil, errors.New("UDB search timeout exceeded on key lookup")
		}
	}

	publicKey := parseGetKey(response)

	return cMixUID, publicKey, nil
}

func hashAndEncode(s string) (string, error) {
	buf := new(bytes.Buffer)
	encoder := base64.NewEncoder(base64.StdEncoding, buf)

	sha := sha256.New()
	sha.Write([]byte(s))
	hashed := sha.Sum(nil)

	_, err := encoder.Write(hashed)
	if err != nil {
		err = errors.New(fmt.Sprintf("Error base64 encoding string %s: %+v", s, err))
		return "", err
	}

	err = encoder.Close()
	if err != nil {
		err = errors.New(fmt.Sprintf("Error closing encoder: %+v", err))
		return "", err
	}

	return buf.String(), nil
}

// parseSearch parses the responses from SEARCH. It returns the user's id and
// the user's public key fingerprint
func parseSearch(msg string) (*id.User, string) {
	globals.Log.DEBUG.Printf("Parsing search response: %v", msg)
	resParts := strings.Split(msg, " ")
	if len(resParts) != 5 {
		return id.ZeroID, fmt.Sprintf("Invalid response from search: %s", msg)
	}

	cMixUIDBytes, err := base64.StdEncoding.DecodeString(resParts[3])
	if err != nil {
		return id.ZeroID, fmt.Sprintf("Couldn't parse search cMix UID: %s", msg)
	}
	cMixUID := id.NewUserFromBytes(cMixUIDBytes)

	return cMixUID, resParts[4]
}

// parseGetKey parses the responses from GETKEY. It returns the
// corresponding public key.
func parseGetKey(msg string) []byte {
	resParts := strings.Split(msg, " ")
	if len(resParts) != 3 {
		globals.Log.WARN.Printf("Invalid response from GETKEY: %s", msg)
		return nil
	}
	keymat, err := base64.StdEncoding.DecodeString(resParts[2])
	if err != nil || len(keymat) == 0 {
		globals.Log.WARN.Printf("Couldn't decode GETKEY keymat: %s", msg)
		return nil
	}
	return keymat
}

// pushKey uploads the users' public key
func pushKey(udbID *id.User, keyFP string, publicKey []byte) error {
	publicKeyString := base64.StdEncoding.EncodeToString(publicKey)
	globals.Log.DEBUG.Printf("Running pushkey for %q, %v, %v", *udbID, keyFP,
		publicKeyString)

	pushKeyMsg := fmt.Sprintf("%s %s", keyFP, publicKeyString)

	return sendCommand(udbID, parse.Pack(&parse.TypedBody{
		MessageType: int32(cmixproto.Type_UDB_PUSH_KEY),
		Body:        []byte(pushKeyMsg),
	}))
}

// keyExists checks for the existence of a key on the bot
func keyExists(udbID *id.User, keyFP string) bool {
	globals.Log.DEBUG.Printf("Running keyexists for %q, %v", *udbID, keyFP)
	cmd := parse.Pack(&parse.TypedBody{
		MessageType: int32(cmixproto.Type_UDB_GET_KEY),
		Body:        []byte(fmt.Sprintf("%s", keyFP)),
	})
	expected := fmt.Sprintf("GETKEY %s NOTFOUND", keyFP)
	err := sendCommand(udbID, cmd)
	if err != nil {
		return false
	}
	getKeyResponse := <-getKeyResponseListener
	if getKeyResponse != expected {
		return true
	}
	return false
}

// fingerprint generates the same fingerprint that the udb should generate
// TODO: Maybe move this helper to crypto module?
func fingerprint(publicKey []byte) string {
	h, _ := hash.NewCMixHash() // why does this return an err and not panic?
	h.Write(publicKey)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}
