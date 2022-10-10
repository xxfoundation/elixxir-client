////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"strings"
	"sync"
)

// errToUserErr maps backend patterns to user-friendly error messages.
// Example format:
// (Back-end) "Building new HostPool because no HostList stored:":  (Front-end) "Missing host list",
var errToUserErr = map[string]string{
	// Registration errors
	//"cannot create username when network is not health" :
	//	"Cannot create username, unable to connect to network",
	//"failed to add due to malformed fact stringified facts must at least have a type at the start" :
	//	"Invalid fact, is the field empty?",
	//// UD failures
	//"failed to create user discovery manager: cannot return single manager, network is not health" :
	//	"Could not connect to user discovery",
	//"user discovery returned error on search: no results found" :
	//	"No results found",
	//"failed to search.: waiting for response to single-use transmisson timed out after 10s" :
	//	"Search timed out",
	//"the phone number supplied was empty" : "Invalid phone number",
	//"failed to create user discovery manager: cannot start ud manager when network follower is not running." :
	//	"Could not get network status",
}

// error<Mux is a global lock for the errToUserErr global.
var errorMux sync.RWMutex

// Error codes
const (
	UnrecognizedCode    = "UR: "
	UnrecognizedMessage = UnrecognizedCode + "Unrecognized error from XX backend, please report"
)

// CreateUserFriendlyErrorMessage will convert the passed in error string to an
// error string that is user-friendly if a substring match is found to a
// common error. Common errors is a map that can be updated using
// UpdateCommonErrors. If the error is not common, some simple parsing is done
// on the error message to make it more user-accessible, removing backend
// specific jargon.
//
// Parameters:
//   - errStr - an error returned from the backend.
//
// Returns
//  - A user-friendly error message. This should be devoid of technical speak
//    but still be meaningful for front-end or back-end teams.
func CreateUserFriendlyErrorMessage(errStr string) string {
	errorMux.RLock()
	defer errorMux.RUnlock()
	// Go through common errors
	for backendErr, userFriendly := range errToUserErr {
		// Determine if error contains a common error
		if strings.Contains(errStr, backendErr) {
			return userFriendly
		}
	}

	descStr := "desc = "
	// If this contains an rpc error, determine how to handle it
	if strings.Contains(errStr, context.DeadlineExceeded.Error()) {
		// If there is a context deadline exceeded message, return the higher level
		// as context deadline exceeded is not informative
		rpcErr := "rpc "
		rpcIdx := strings.Index(errStr, rpcErr)
		return errStr[:rpcIdx]
	} else if strings.Contains(errStr, descStr) {
		// If containing an rpc error where context deadline exceeded
		// is NOT involved, the error returned server-side is often
		//more informative
		descIdx := strings.Index(errStr, descStr)
		// return everything after "desc = "
		return errStr[descIdx+len(descStr):]
	}

	// If a compound error message, return the highest level message
	errParts := strings.Split(errStr, ":")
	if len(errParts) > 1 {
		// Return everything before the first :
		return UnrecognizedCode + errParts[0]
	}

	return fmt.Sprintf("%s: %v", UnrecognizedCode, errStr)
}

// UpdateCommonErrors updates the internal error mapping database. This internal
// database maps errors returned from the backend to user-friendly error
// messages.
//
// Parameters:
//  - jsonFile - contents of a JSON file whose format conforms to the example below.
//
// Example Input:
//  {
//    "Failed to Unmarshal Conversation": "Could not retrieve conversation",
//    "Failed to unmarshal SentRequestMap": "Failed to pull up friend requests",
//    "cannot create username when network is not health": "Cannot create username, unable to connect to network",
//  }
func UpdateCommonErrors(jsonFile string) error {
	errorMux.Lock()
	defer errorMux.Unlock()
	err := json.Unmarshal([]byte(jsonFile), &errToUserErr)
	if err != nil {
		return errors.WithMessage(err, "Failed to unmarshal json file, "+
			"did you pass in the contents or the path?")
	}

	return nil
}
