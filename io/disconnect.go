////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"gitlab.com/privategrity/comms/connect"
)

// Disconnect from the server
func Disconnect(address string) {
	connect.Disconnect(address)
}
