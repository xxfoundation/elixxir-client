////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package store

type RequestType uint

const (
	Sent    RequestType = 1
	Receive RequestType = 2
)

type requestDisk struct {
	T  uint
	ID []byte
}
