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
	"gitlab.com/privategrity/client/globals"
	"gitlab.com/privategrity/client/io"
	"gitlab.com/privategrity/client/parse"
	"gitlab.com/privategrity/client/switchboard"
	"gitlab.com/privategrity/client/user"
	"gitlab.com/privategrity/crypto/hash"
	"strconv"
	"strings"
)

// UdbID is the ID of the user discovery bot, which is always 13
const udbID = user.ID(13)

type udbResponseListener chan string

var pushKeyResponseListener udbResponseListener
var getKeyResponseListener udbResponseListener
var registerResponseListener udbResponseListener
var searchResponseListener udbResponseListener

func (l *udbResponseListener) Hear(msg *parse.Message,
	isHeardElsewhere bool) {
	*l <- string(msg.Body)
}

// The go runtime calls init() before calling any methods in the package
func init() {
	pushKeyResponseListener = make(udbResponseListener)
	getKeyResponseListener = make(udbResponseListener)
	registerResponseListener = make(udbResponseListener)
	searchResponseListener = make(udbResponseListener)

	switchboard.Listeners.Register(udbID, parse.Type_UDB_PUSH_KEY_RESPONSE,
		&pushKeyResponseListener)
	switchboard.Listeners.Register(udbID, parse.Type_UDB_GET_KEY_RESPONSE,
		&getKeyResponseListener)
	switchboard.Listeners.Register(udbID, parse.Type_UDB_REGISTER_RESPONSE,
		&registerResponseListener)
	switchboard.Listeners.Register(udbID, parse.Type_UDB_SEARCH_RESPONSE,
		&searchResponseListener)
}

// Register sends a registration message to the UDB. It does this by sending 2
// PUSHKEY messages to the UDB, then calling UDB's REGISTER command.
// If any of the commands fail, it returns an error.
func Register(valueType, value string, publicKey []byte) error {
	keyFP := fingerprint(publicKey)

	// check if key already exists and push one if it doesn't
	if !keyExists(udbID, keyFP) {
		err := pushKey(udbID, keyFP, publicKey)
		if err != nil {
			return fmt.Errorf("Could not PUSHKEY: %s", err.Error())
		}
	}

	msgBody := parse.Pack(&parse.TypedBody{
		Type: parse.Type_UDB_REGISTER,
		Body: []byte(fmt.Sprintf("%s %s %s", valueType, value, keyFP)),
	})

	// Send register command
	err := sendCommand(udbID, msgBody)
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
func Search(valueType, value string) (map[uint64][]byte, error) {
	msgBody := parse.Pack(&parse.TypedBody{
		Type: parse.Type_UDB_SEARCH,
		Body: []byte(fmt.Sprintf("%s %s", valueType, value)),
	})
	err := sendCommand(udbID, msgBody)
	if err != nil {
		return nil, err
	}
	response := <-searchResponseListener
	empty := fmt.Sprintf("SEARCH %s NOTFOUND", value)
	if response == empty {
		return nil, nil
	}
	// While search returns more than 1 result, we only process the first
	cMixUID, keyFP := parseSearch(response)
	if cMixUID == 0 {
		return nil, fmt.Errorf("%s", keyFP)
	}

	// Get the full key and decode it
	msgBody = parse.Pack(&parse.TypedBody{
		Type: parse.Type_UDB_GET_KEY,
		Body: []byte(keyFP),
	})
	err = sendCommand(udbID, msgBody)
	if err != nil {
		return nil, err
	}
	response = <-getKeyResponseListener
	publicKey := parseGetKey(response)

	actualFP := fingerprint(publicKey)
	if keyFP != actualFP {
		return nil, fmt.Errorf("Fingerprint for %s did not match %s!", keyFP,
			actualFP)
	}

	retval := make(map[uint64][]byte)
	retval[cMixUID] = publicKey

	return retval, nil
}

// parseSearch parses the responses from SEARCH. It returns the user's id and
// the user's public key fingerprint
func parseSearch(msg string) (uint64, string) {
	resParts := strings.Split(msg, " ")
	if len(resParts) != 5 {
		return 0, fmt.Sprintf("Invalid response from search: %s", msg)
	}

	cMixUID, err := strconv.ParseUint(resParts[3], 10, 64)
	if err != nil {
		return 0, fmt.Sprintf("Couldn't parse search cMix UID: %s", msg)
	}

	return cMixUID, resParts[4]
}

// parseGetKey parses the responses from GETKEY. It returns the
// corresponding public key.
func parseGetKey(msg string) []byte {
	resParts := strings.Split(msg, " ")
	if len(resParts) != 3 {
		globals.N.WARN.Printf("Invalid response from GETKEY: %s", msg)
		return nil
	}
	keymat, err := base64.StdEncoding.DecodeString(resParts[2])
	if err != nil || len(keymat) != 256 {
		globals.N.WARN.Printf("Couldn't decode GETKEY keymat: %s", msg)
		return nil
	}

	return keymat
}

// pushKey uploads the users' public key
func pushKey(udbID user.ID, keyFP string, publicKey []byte) error {
	publicKeyString := base64.StdEncoding.EncodeToString(publicKey)
	expected := fmt.Sprintf("PUSHKEY COMPLETE %s", keyFP)

	sendCommand(udbID, parse.Pack(&parse.TypedBody{
		Type: parse.Type_UDB_PUSH_KEY,
		Body: []byte(fmt.Sprintf("%s %s", keyFP, publicKeyString)),
	}))
	response := <-pushKeyResponseListener
	if response != expected {
		return fmt.Errorf("PUSHKEY Failed: %s", response)
	}
	return nil
}

// keyExists checks for the existence of a key on the bot
func keyExists(udbID user.ID, keyFP string) bool {
	cmd := parse.Pack(&parse.TypedBody{
		Type: parse.Type_UDB_GET_KEY,
		Body: []byte(fmt.Sprintf("%s", keyFP)),
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

// sendCommand sends a command to the udb. This doesn't block.
// Callers that need to wait on a response should implement waiting with a
// listener.
func sendCommand(botID user.ID, command []byte) error {
	return io.Messaging.SendMessage(botID, string(command))
}
