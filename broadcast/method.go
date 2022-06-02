////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package broadcast

// Method enum for broadcast type
type Method uint8

const (
	Symmetric Method = iota
	Asymmetric
)

func (m Method) String() string {
	switch m {
	case Symmetric:
		return "Symmetric"
	case Asymmetric:
		return "Asymmetric"
	default:
		return "Unknown"
	}
}
