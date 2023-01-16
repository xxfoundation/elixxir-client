////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

// NOTE: file access not available in wasm, and this reads test keys from
// the comms repo...

//go:build !js || !wasm
// +build !js !wasm

package gateway

import (
	"bytes"
	"testing"

	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/comms/testkeys"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/primitives/id"
	"golang.org/x/crypto/blake2b"
)

// Test load & store functions for cert checker
func Test_certChecker_loadStore(t *testing.T) {
	kv := versioned.NewKV(ekv.MakeMemstore())
	cc := newCertChecker(&mockCertCheckerComm{}, kv)

	// FIXME: This should load from a variable not disk.
	gwCert := testkeys.LoadFromPath(testkeys.GetGatewayCertPath())
	gwID := id.NewIdFromString("testid01", id.Gateway, t)

	expectedFp := blake2b.Sum256(gwCert)

	fp, err := cc.loadGatewayCertificateFingerprint(gwID)
	if err == nil || fp != nil {
		t.Errorf("Should error & receive nil when nothing is in storage")
	}

	err = cc.storeGatewayCertificateFingerprint(expectedFp[:], gwID)
	if err != nil {
		t.Fatal("Failed to store certificate")
	}

	fp, err = cc.loadGatewayCertificateFingerprint(gwID)
	if err != nil {
		t.Fatalf("Failed to load certificate for gwID %s: %+v", gwID, err)
	}

	if bytes.Compare(fp, expectedFp[:]) != 0 {
		t.Errorf("Did not receive expected fingerprint after load\n\tExpected: %+v\n\tReceived: %+v\n", expectedFp, fp)
	}
}
