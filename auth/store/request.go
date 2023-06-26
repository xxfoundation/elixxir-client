////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package store

import "gitlab.com/xx_network/primitives/id"

type RequestType uint

const (
	Sent    RequestType = 1
	Receive RequestType = 2
)

type requestDisk struct {
	T  RequestType `json:"t"`
	ID *id.ID      `json:"ID"`
}
