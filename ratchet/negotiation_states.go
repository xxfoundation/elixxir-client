////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package ratchet

import (
	"fmt"
)

// Fix-me: this solution is incompatible with offline sending, when that is
// added, a session which has not been confirmed will never partnerSource the
// creation of new session, the Unconfirmed->Confirmed and
// Confirmed->NewSessionCreated most likely need to be two separate enums
// tracked separately
type NegotiationState uint8

const (
	Unconfirmed NegotiationState = iota
	Sending
	Sent
	Confirmed
	NewSessionTriggered
	NewSessionCreated
)

var legalStateChanges = [][]bool{
	// Unconf  Sending  Sent   Confi  NewTrig  NewCreat
	{false, true, false, false, false, false}, // Unc
	{true, false, true, true, false, false},   // Sending
	{false, false, false, true, false, false}, // Sent
	{false, false, false, false, true, false}, // Confi
	{false, false, false, true, false, true},  // NewTrig
	{false, false, true, false, false, false}, // NewCreat
}

func (n NegotiationState) IsNewStateLegal(newState NegotiationState) bool {
	return legalStateChanges[n][newState]
}

// Adherence to stringer interface
func (n NegotiationState) String() string {
	switch n {
	case Unconfirmed:
		return "Unconfirmed"
	case Sending:
		return "Sending"
	case Sent:
		return "Sent"
	case Confirmed:
		return "Confirmed"
	case NewSessionTriggered:
		return "NewSessionTriggered"
	case NewSessionCreated:
		return "NewSessionCreated"
	default:
		return fmt.Sprintf("Unknown NegotiationState %v", uint8(n))
	}
}
