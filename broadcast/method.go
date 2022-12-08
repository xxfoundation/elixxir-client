////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package broadcast

import "strconv"

// Method enum for broadcast type.
type Method uint8

const (
	// Symmetric messages can be sent to anyone in the broadcast.
	Symmetric Method = iota
	RSAToPublic
	RSAToPrivate
)

// String prints a human-readable string representation of the Method used for
// logging and debugging. This function adheres to the fmt.Stringer interface.
func (m Method) String() string {
	switch m {
	case Symmetric:
		return "Symmetric"
	case RSAToPublic:
		return "RSAToPublic"
	case RSAToPrivate:
		return "RSAToPrivate"
	default:
		return "INVALID METHOD " + strconv.Itoa(int(m))
	}
}
