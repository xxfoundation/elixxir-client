///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

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

////// Error codes //////

const UnrecognizedCode = "UR: "
const UnrecognizedMessage = UnrecognizedCode + "Unrecognized error from XX backend, please report"

// ErrorStringToUserFriendlyMessage takes a passed in errStr which will be
// a backend generated error. These may be error specifically written by
// the backend team or lower level errors gotten from low level dependencies.
// This function will parse the error string for common errors provided from
// errToUserErr to provide a more user-friendly error message for the front end.
// If the error is not common, some simple parsing is done on the error message
// to make it more user-accessible, removing backend specific jargon.
func ErrorStringToUserFriendlyMessage(errStr string) string {
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

// UpdateCommonErrors takes the passed in contents of a JSON file and updates the
// errToUserErr map with the contents of the json file. The JSON's expected format
// conform with the commented examples provides in errToUserErr above.
// NOTE that you should not pass in a file path, but a preloaded JSON file
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
