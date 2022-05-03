////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package groupChat

import (
	"fmt"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/primitives/format"
)

type Processor interface {
	// Process decrypts and hands off the message to its internal down stream
	// message processing system.
	Process(decryptedMsg MessageReceive, msg format.Message,
		receptionID receptionID.EphemeralIdentity, round rounds.Round)

	// Stringer interface for debugging
	fmt.Stringer
}
