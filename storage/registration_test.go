////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package storage

import (
	"bytes"
	"gitlab.com/elixxir/ekv"
	"testing"
)

func TestSession_RegState(t *testing.T) {
	store := make(ekv.Memstore)
	testSession := &Session{NewVersionedKV(store)}

	expectedVal := int64(42)
	err := testSession.SetRegState(expectedVal)
	if err != nil {
		t.Errorf("Failed to place value in session: %v", err)
	}

	retrievedVal, err := testSession.GetRegState()
	if err != nil {
		t.Errorf("Faield to get value from session: %v", err)
	}

	if retrievedVal != expectedVal {
		t.Errorf("Expected value not retrieved from file store!"+
			"\n\tExpected: %v"+
			"\n\tRecieved: %v", expectedVal, retrievedVal)
	}

}

func TestSession_RegValidation(t *testing.T) {
	store := make(ekv.Memstore)
	testSession := &Session{NewVersionedKV(store)}

	expectedVal := []byte("testData")

	err := testSession.SetRegValidationSig(expectedVal)
	if err != nil {
		t.Errorf("Failed to place value in session: %v", err)
	}

	retrievedVal, err := testSession.GetRegValidationSig()
	if err != nil {
		t.Errorf("Faield to get value from session: %v", err)
	}

	if !bytes.Equal(retrievedVal, expectedVal) {
		t.Errorf("Expected value not retrieved from file store!"+
			"\n\tExpected: %v"+
			"\n\tRecieved: %v", expectedVal, retrievedVal)
	}
}
