package pickup

import (
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/primitives/id"
)

type RoundGetter interface {
	GetRound(id id.Round) (*pb.RoundInfo, error)
}
