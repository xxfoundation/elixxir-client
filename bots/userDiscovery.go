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
	"strconv"
	"strings"
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

// Search returns a userID and public key based on the search criteria
func Search(valueType, value string) (map[uint64][]byte, error) {
	response := sendCommand(udbID, fmt.Sprintf("SEARCH %s %s", valueType, value))
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
	responses := sendCommandMulti(2, udbID, fmt.Sprintf("GETKEY %s", keyFP))
	publicKey := make([]byte, 256)
	for i := 0; i < 2; i++ {
		idx, keymat := parseGetKey(responses[i])
		for j := range keymat {
			publicKey[j+idx] = keymat[j]
		}

	}

	actualFP := fingerprint(publicKey)
	if keyFP != actualFP {
		return nil, fmt.Errorf("Fingerprint for %s did not match %s!", keyFP,
			actualFP)
	}

	retval := make(map[uint64][]byte)
	retval[cMixUID] = publicKey

	return retval, nil
}

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

func parseGetKey(msg string) (int, []byte) {
	resParts := strings.Split(msg, " ")
	if len(resParts) != 4 {
		jww.WARN.Printf("Invalid response from GETKEY: %s", msg)
		return -1, nil
	}

	idx, err := strconv.ParseInt(resParts[2], 10, 32)
	if err != nil {
		jww.WARN.Printf("Couldn't parse GETKEY Index: %s", msg)
		return -1, nil
	}
	keymat, err := base64.StdEncoding.DecodeString(resParts[3])
	if err != nil || len(keymat) != 128 {
		jww.WARN.Printf("Couldn't decode GETKEY keymat: %s", msg)
		return -1, nil
	}

	return int(idx), keymat
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

func sendCommandMulti(responseCnt int, botID uint64, command string) []string {
	listener := io.Messaging.Listen(botID)
	defer io.Messaging.StopListening(listener)
	err := io.Messaging.SendMessage(botID, command)

	responses := make([]string, 0)
	if err != nil {
		responses = append(responses, err.Error())
		return responses
	}

	for i := 0; i < responseCnt; i++ {
		response := <-listener
		responses = append(responses, response.GetPayload())
	}
	return responses
}
