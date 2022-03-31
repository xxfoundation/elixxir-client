package session

import "fmt"

type Params struct {
	// using the DH as a seed, both sides finalizeKeyNegotation a number
	// of keys to use before they must rekey because
	// there are no keys to use.
	MinKeys uint16
	MaxKeys uint16
	// the percent of keys before a rekey is attempted. must be <0
	RekeyThreshold float64
	// extra keys generated and reserved for rekey attempts. This
	// many keys are not allowed to be used for sending messages
	// in order to ensure there are extras for rekeying.
	NumRekeys uint16
	// Number from 0 to 1, denotes how often when in the unconfirmed state the
	// system will automatically resend the rekey request on any message send
	// from the partner the session is associated with
	UnconfirmedRetryRatio float64
}

// DEFAULT KEY GENERATION PARAMETERS
// Hardcoded limits for keys
// sets the number of keys very high, but with a low rekey threshold. In this case, if the other party is online, you will read
const (
	minKeys       uint16  = 1000
	maxKeys       uint16  = 2000
	rekeyThrshold float64 = 0.05
	numReKeys     uint16  = 16
	rekeyRatio    float64 = 1 / 10
)

func GetDefaultE2ESessionParams() Params {
	return Params{
		MinKeys:               minKeys,
		MaxKeys:               maxKeys,
		RekeyThreshold:        rekeyThrshold,
		NumRekeys:             numReKeys,
		UnconfirmedRetryRatio: rekeyRatio,
	}
}

func (p Params) String() string {
	return fmt.Sprintf("SessionParams{ MinKeys: %d, MaxKeys: %d, NumRekeys: %d }",
		p.MinKeys, p.MaxKeys, p.NumRekeys)
}
