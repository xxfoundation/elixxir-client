////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package broadcast

// Method enum for broadcast type.
type Method uint8

const (
	Symmetric Method = iota
	RSAToPublic
	RSAToPrivate
)

func (m Method) String() string {
	switch m {
	case Symmetric:
		return "Symmetric"
	case RSAToPublic:
		return "RSAToPublic"
	case RSAToPrivate:
		return "RSAToPrivate"
	default:
		return "Unknown"
	}
}
