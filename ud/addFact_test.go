////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

// NOTE: ud is not available in wasm
//go:build !js || !wasm

package ud

import (
	"os"
	"testing"

	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
)

func TestMain(m *testing.M) {
	jww.SetStdoutThreshold(jww.LevelTrace)
	connect.TestingOnlyDisableTLS = true
	os.Exit(m.Run())
}

type testAFC struct{}

// Dummy implementation of SendRegisterFact so that we don't need to run our own
// UDB server.
func (rFC *testAFC) SendRegisterFact(*connect.Host, *pb.FactRegisterRequest) (
	*pb.FactRegisterResponse, error) {
	return &pb.FactRegisterResponse{}, nil
}

// Test that the addFact function completes successfully
func TestAddFact(t *testing.T) {

	m, _ := newTestManager(t)

	// Create our test fact
	USCountryCode := "US"
	USNumber := "6502530000"
	f := fact.Fact{
		Fact: USNumber + USCountryCode,
		T:    2,
	}

	// Set up a dummy comms that implements SendRegisterFact
	// This way we don't need to run UDB just to check that this
	// function works.
	tAFC := testAFC{}
	uid := &id.ID{}
	// Run addFact and see if it returns without an error!
	_, err := m.addFact(f, uid, &tAFC)
	if err != nil {
		t.Fatal(err)
	}
}
