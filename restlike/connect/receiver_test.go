////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package connect

import (
	"gitlab.com/elixxir/client/e2e/receive"
	"gitlab.com/elixxir/client/restlike"
	"testing"
)

// Test failure of proto unmarshal
func TestSingleReceiver_Callback_FailUnmarshal(t *testing.T) {
	ep := restlike.NewEndpoints()
	r := receiver{endpoints: ep}

	r.Hear(receive.Message{Payload: []byte("test")})
}
