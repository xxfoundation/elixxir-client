///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package ud

import (
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/comms/connect"
	"testing"
)

type testUsernameValidation struct{}

func (tuv *testUsernameValidation) SendUsernameValidation(host *connect.Host,
	message *pb.UsernameValidationRequest) (*pb.UsernameValidation, error) {
	return &pb.UsernameValidation{}, nil
}

func TestManager_GetUsernameValidationSignature(t *testing.T) {
	// Create our Manager object
	m, _ := newTestManager(t)
	c := &testUsernameValidation{}

	_, err := m.getUsernameValidationSignature("admin", c)
	if err != nil {
		t.Fatalf("getUsernameValidationSignature error: %+v", err)
	}

}
