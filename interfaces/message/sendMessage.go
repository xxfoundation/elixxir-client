///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package message

import "gitlab.com/xx_network/primitives/id"

type Send struct {
	Recipient   *id.ID
	Payload     []byte
	MessageType Type
}
