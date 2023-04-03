////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package groupChat

import (
	"fmt"
	"gitlab.com/elixxir/client/v4/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/primitives/format"
)

// Processor manages the handling of received group chat messages.
type Processor interface {
	// Process decrypts and hands off the message to its internal down stream
	// message processing system.
	Process(decryptedMsg MessageReceive, msg format.Message, _ []string, _ []byte,
		receptionID receptionID.EphemeralIdentity, round rounds.Round)

	// Stringer interface for debugging.
	fmt.Stringer
}
