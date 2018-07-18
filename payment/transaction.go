////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package payment

import (
	"gitlab.com/privategrity/crypto/coin"
	"gitlab.com/privategrity/client/user"
	"time"
)

type Transaction struct{
	Create 	coin.Sleeve
	Destroy	[]coin.Sleeve
	Change 	coin.Sleeve

	Sender 		user.ID
	Recipient 	user.ID

	Description string

	Timestamp 	time.Time

	Value uint64
}