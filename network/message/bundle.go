package message

import (
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/xx_network/primitives/id"
)

type Bundle struct {
	Round    id.Round
	Messages []format.Message
	Finish   func()
}
