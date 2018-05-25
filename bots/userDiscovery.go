////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Package bot functions for working with the user discovery bot (UDB)
package bot

import (
	"encoding/base64"
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/privategrity/client/io"
	"gitlab.com/privategrity/crypto/hash"
)

// UdbID is the ID of the user discovery bot, which is always 13
const udbID = uint64(13)

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

	// Send register command
	regResult := sendCommand(udbID, fmt.Sprintf("REGISTER %s %s %s",
		valueType, value, keyFP))
	if regResult != "REGISTRATION COMPLETE" {
		return fmt.Errorf("Registration failed: %s", regResult)
	}
	return nil
}

// Returns a list of users matching the search criteria, or nil if
// none were found.
func Search(valueType, value string) []string {
	return nil
}

func pushKey(udbID uint64, keyFP string, publicKey []byte) error {
	publicKeyParts := make([]string, 2)
	publicKeyParts[0] = base64.StdEncoding.EncodeToString(publicKey[:128])
	publicKeyParts[1] = base64.StdEncoding.EncodeToString(publicKey[128:])
	cmd := "PUSHKEY %s %d %s"
	expected := fmt.Sprintf("PUSHKEY COMPLETE %s", keyFP)
	sendCommand(udbID, fmt.Sprintf(cmd, keyFP, 0, publicKeyParts[0]))
	r := sendCommand(udbID, fmt.Sprintf(cmd, keyFP, 128, publicKeyParts[1]))
	if r != expected {
		return fmt.Errorf("PUSHKEY Failed: %s", r)
	}
	return nil
}

func keyExists(udbID uint64, keyFP string) bool {
	cmd := fmt.Sprintf("GETKEY %s", keyFP)
	expected := fmt.Sprintf("GETKEY %s NOTFOUND", keyFP)
	getKeyResponse := sendCommand(udbID, cmd)
	return getKeyResponse != expected
}

// fingerprint generates the same fingerprint that the udb should generate
// TODO: Maybe move this helper to crypto module?
func fingerprint(publicKey []byte) string {
	h, _ := hash.NewCMixHash() // why does this return an err and not panic?
	h.Write(publicKey)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// sendCommand sends a command to the udb. This can block forever, but
// only does so if the send command succeeds. Our assumption is that
// we will eventually receive a response from the server. Callers
// to registration that need timeouts should implement it themselves.
func sendCommand(botID uint64, command string) string {
	listener := io.Messaging.Listen(botID)
	defer io.Messaging.StopListening(listener)
	err := io.Messaging.SendMessage(botID, command)
	if err != nil {
		return err.Error()
	}
	response := <-listener
	jww.ERROR.Printf(response.GetPayload())
	return response.GetPayload()
}
