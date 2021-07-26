///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"context"
	"strings"
)

// ErrToUserErr maps backend patterns to user friendly error messages.
// Example format:
// (Back-end) "Building new HostPool because no HostList stored:":  (Front-end) "Missing host list",
var ErrToUserErr = map[string]string{
	// todo populate with common errors
}

// Error codes
const UnrecognizedCode = "UR: "
const UnrecognizedMessage = UnrecognizedCode + "Unrecognized error from XX backend, please report"

// ErrorStringToUserFriendlyMessage takes a passed in errStr which will be
// a backend generated error. These may be error specifically written by
// the backend team or lower level errors gotten from low level dependencies.
// This function will parse the error string for common errors provided from
// ErrToUserErr to provide a more user-friendly error message for the front end.
// If the error is not common, some simple parsing is done on the error message
// to make it more user-accessible, removing backend specific jargon.
func ErrorStringToUserFriendlyMessage(errStr string) string {
	// Go through common errors
	for backendErr, userFriendly := range ErrToUserErr {
		// Determine if error contains a common error
		// Fixme: later versions may be improved by using regex
		if strings.HasPrefix(errStr, backendErr) {
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
		return UnrecognizedCode + errStr[:rpcIdx]
	} else if strings.Contains(errStr, descStr) {
		// If containing an rpc error where context deadline exceeded
		// is NOT involved, the error returned server-side is often
		//more informative
		descIdx := strings.Index(errStr, descStr)
		// return everything after "desc = "
		return UnrecognizedCode + errStr[descIdx+len(descStr):]
	}

	// If a compound error message, return the highest level message
	errParts := strings.Split(errStr, ":")
	if len(errParts) > 1 {
		// Return everything before the first :
		return UnrecognizedCode + errParts[0]
	}

	return UnrecognizedMessage
}
