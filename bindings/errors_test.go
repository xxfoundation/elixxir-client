////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"context"
	"strings"
	"testing"
)

// Unit test
func TestErrorStringToUserFriendlyMessage(t *testing.T) {
	// Setup: Populate map
	backendErrs := []string{"Failed to Unmarshal Conversation", "failed to create group key preimage",
		"Failed to unmarshal SentRequestMap"}
	userErrs := []string{"Could not retrieve conversation", "Failed to initiate group chat",
		"Failed to pull up friend requests"}

	for i, exampleErr := range backendErrs {
		errToUserErr[exampleErr] = userErrs[i]
	}

	// Check if a mapped common error returns the expected user-friendly error
	received := CreateUserFriendlyErrorMessage(backendErrs[0])
	if strings.Compare(received, userErrs[0]) != 0 {
		t.Errorf("Unexpected user friendly message returned from common error mapping."+
			"\n\tExpected: %s"+
			"\n\tReceived: %v", userErrs[0], received)
	}

	// Test RPC error in which high level information should
	// be passed along (ie context deadline exceeded error)
	expected := "Could not poll network: "
	rpcPrefix := "rpc error: desc = "
	rpcErr := expected + rpcPrefix + context.DeadlineExceeded.Error()
	received = CreateUserFriendlyErrorMessage(rpcErr)
	if strings.Compare(expected, received) != 0 {
		t.Errorf("Rpc error parsed unxecpectedly with error "+
			"\n\"%s\" "+
			"\n\tExpected: %s"+
			"\n\tReceived: %v", rpcErr, UnrecognizedCode+expected, received)
	}

	// Test RPC error where server side error information is provided
	serverSideError := "Could not parse message! Please try again with a properly crafted message"
	rpcErr = rpcPrefix + serverSideError
	received = CreateUserFriendlyErrorMessage(rpcErr)
	if strings.Compare(serverSideError, received) != 0 {
		t.Errorf("RPC error parsed unexpectedly with error "+
			"\n\"%s\" "+
			"\n\tExpected: %s"+
			"\n\tReceived: %v", rpcErr, UnrecognizedCode+serverSideError, received)
	}

	// Test uncommon error, should return highest level message
	expected = "failed to register with permissioning"
	uncommonErr := expected + ": sendRegistrationMessage: Unable to contact Identity Server"
	received = CreateUserFriendlyErrorMessage(uncommonErr)
	if strings.Compare(received, UnrecognizedCode+expected) != 0 {
		t.Errorf("Uncommon error parsed unexpectedly with error "+
			"\n\"%s\" "+
			"\n\tExpected: %s"+
			"\n\tReceived: %s", uncommonErr, UnrecognizedCode+expected, received)
	}

	// Test fully unrecognizable and un-parsable message,
	// should hardcoded error message
	uncommonErr = "failed to register with permissioning"
	received = CreateUserFriendlyErrorMessage(uncommonErr)
	if strings.Compare(UnrecognizedCode+": "+uncommonErr, received) != 0 {
		t.Errorf("Uncommon error parsed unexpectedly with error "+
			"\n\"%s\" "+
			"\n\tExpected: %s"+
			"\n\tReceived: %s", uncommonErr, UnrecognizedMessage, received)
	}

}

// Unit test
func TestClient_UpdateCommonErrors(t *testing.T) {

	key, expectedVal := "failed to create group key preimage", "Failed to initiate group chat"

	jsonData := "{\"Failed to Unmarshal Conversation\":\"Could not retrieve conversation\",\"Failed to unmarshal SentRequestMap\":\"Failed to pull up friend requests\",\"failed to create group key preimage\":\"Failed to initiate group chat\"}\n"

	err := UpdateCommonErrors(jsonData)
	if err != nil {
		t.Fatalf("UpdateCommonErrors error: %v", err)
	}

	val, ok := errToUserErr[key]
	if !ok {
		t.Fatalf("Expected entry was not populated")
	}

	if strings.Compare(expectedVal, val) != 0 {
		t.Fatalf("Entry in updated error map was not expected."+
			"\n\tExpected: %s"+
			"\n\tReceived: %s", expectedVal, val)
	}

}
