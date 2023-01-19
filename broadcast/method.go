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
	// Symmetric messages can be sent by anyone in the channel to everyone else
	// in the channel.
	//
	//  all -> all
	Symmetric Method = iota

	// RSAToPublic messages can be sent from the channel admin to everyone else
	// in the channel.
	//
	//  admin -> all
	RSAToPublic

	// RSAToPrivate messages can be sent from anyone in the channel to the
	// channel admin.
	//
	//  all -> admin
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
