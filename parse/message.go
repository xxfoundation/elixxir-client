////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package parse

import (
	"gitlab.com/privategrity/client/user"
)

type Message struct {
	TypedBody
	Sender   user.ID
	Receiver user.ID
}
