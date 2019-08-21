////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Package bot functions for working with the user discovery bot (UDB)
package bots

import (
	"encoding/base64"
	"fmt"
	"gitlab.com/elixxir/client/cmixproto"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/primitives/id"
	"strings"
)

// Register sends a registration message to the UDB. It does this by sending 2
// PUSHKEY messages to the UDB, then calling UDB's REGISTER command.
// If any of the commands fail, it returns an error.
// valueType: Currently only "EMAIL"
func Register(valueType, value string, publicKey []byte) error {
	globals.Log.DEBUG.Printf("Running register for %v, %v, %q", valueType,
		value, publicKey)
	keyFP := fingerprint(publicKey)

	// check if key already exists and push one if it doesn't
	if !keyExists(UdbID, keyFP) {
		err := pushKey(UdbID, keyFP, publicKey)
		if err != nil {
			return fmt.Errorf("Could not PUSHKEY: %s", err.Error())
		}
	}

	msgBody := parse.Pack(&parse.TypedBody{
		MessageType: int32(cmixproto.Type_UDB_REGISTER),
		Body:        []byte(fmt.Sprintf("%s %s %s", valueType, value, keyFP)),
	})

	// Send register command
	err := sendCommand(UdbID, msgBody)
	if err == nil {
		regResult := <-registerResponseListener
		if regResult != "REGISTRATION COMPLETE" {
			return fmt.Errorf("Registration failed: %s", regResult)
		}
		return nil
	} else {
		return err
	}
}

// Search returns a userID and public key based on the search criteria
// it accepts a valueType of EMAIL and value of an e-mail address, and
// returns a map of userid -> public key
func Search(valueType, value string) (*id.User, []byte, error) {
	globals.Log.DEBUG.Printf("Running search for %v, %v", valueType, value)
	msgBody := parse.Pack(&parse.TypedBody{
		MessageType: int32(cmixproto.Type_UDB_SEARCH),
		Body:        []byte(fmt.Sprintf("%s %s", valueType, value)),
	})
	err := sendCommand(UdbID, msgBody)
	if err != nil {
		return nil, nil, err
	}
	response := <-searchResponseListener
	empty := fmt.Sprintf("SEARCH %s NOTFOUND", value)
	if response == empty {
		return nil, nil, nil
	}
	// While search returns more than 1 result, we only process the first
	cMixUID, keyFP := parseSearch(response)
	if *cMixUID == *id.ZeroID {
		return nil, nil, fmt.Errorf("%s", keyFP)
	}

	// Get the full key and decode it
	msgBody = parse.Pack(&parse.TypedBody{
		MessageType: int32(cmixproto.Type_UDB_GET_KEY),
		Body:        []byte(keyFP),
	})
	err = sendCommand(UdbID, msgBody)
	if err != nil {
		return nil, nil, err
	}
	response = <-getKeyResponseListener
	publicKey := parseGetKey(response)

	actualFP := fingerprint(publicKey)
	if keyFP != actualFP {
		return nil, nil, fmt.Errorf("Fingerprint for %s did not match %s!",
			keyFP,
			actualFP)
	}

	return cMixUID, publicKey, nil
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
	expected := fmt.Sprintf("PUSHKEY COMPLETE %s", keyFP)

	sendCommand(udbID, parse.Pack(&parse.TypedBody{
		MessageType: int32(cmixproto.Type_UDB_PUSH_KEY),
		Body:        []byte(fmt.Sprintf("%s %s", keyFP, publicKeyString)),
	}))
	response := <-pushKeyResponseListener
	if response != expected {
		return fmt.Errorf("PUSHKEY Failed: %s", response)
	}
	return nil
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
