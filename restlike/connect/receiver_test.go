////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package connect

import (
	"gitlab.com/elixxir/client/v4/e2e/receive"
	"gitlab.com/elixxir/client/v4/restlike"
	"testing"
)

// Test failure of proto unmarshal
func TestSingleReceiver_Callback_FailUnmarshal(t *testing.T) {
	ep := restlike.NewEndpoints()
	r := receiver{endpoints: ep}

	r.Hear(receive.Message{Payload: []byte("test")})
}
