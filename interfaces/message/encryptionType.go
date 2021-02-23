///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package message

type EncryptionType uint8

const (
	None EncryptionType = 0
	E2E  EncryptionType = 1
)

func (et EncryptionType)String()string{
	switch et{
	case None:
		return "None"
	case E2E:
		return "E2E"
	default:
		return "Unknown"
	}
}