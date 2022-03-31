///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package auth

type RequestType uint

const (
	Sent    RequestType = 1
	Receive RequestType = 2
)

type requestDisk struct {
	T    uint
	ID   []byte
	MyID []byte
}
