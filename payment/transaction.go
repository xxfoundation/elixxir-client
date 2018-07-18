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

	From 	user.ID
	To 		user.ID

	Description string

	Timestamp 	time.Time

	Value uint64
}

