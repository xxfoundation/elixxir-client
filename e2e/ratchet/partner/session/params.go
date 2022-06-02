package session

import (
	"encoding/json"
	"fmt"
)

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

// paramsDisk will be the marshal-able and umarshal-able object.
type paramsDisk struct {
	MinKeys               uint16
	MaxKeys               uint16
	RekeyThreshold        float64
	NumRekeys             uint16
	UnconfirmedRetryRatio float64
}

// GetDefaultParams returns a default set of Params.
func GetDefaultParams() Params {
	return Params{
		MinKeys:               minKeys,
		MaxKeys:               maxKeys,
		RekeyThreshold:        rekeyThrshold,
		NumRekeys:             numReKeys,
		UnconfirmedRetryRatio: rekeyRatio,
	}
}

// GetParameters returns the default Params, or override with given
// parameters, if set.
func GetParameters(params string) (Params, error) {
	p := GetDefaultParams()
	if len(params) > 0 {
		err := json.Unmarshal([]byte(params), &p)
		if err != nil {
			return Params{}, err
		}
	}
	return p, nil
}

// MarshalJSON adheres to the json.Marshaler interface.
func (p Params) MarshalJSON() ([]byte, error) {
	pDisk := paramsDisk{
		MinKeys:               p.MinKeys,
		MaxKeys:               p.MaxKeys,
		RekeyThreshold:        p.RekeyThreshold,
		NumRekeys:             p.NumRekeys,
		UnconfirmedRetryRatio: p.UnconfirmedRetryRatio,
	}
	return json.Marshal(&pDisk)

}

// UnmarshalJSON adheres to the json.Unmarshaler interface.
func (p *Params) UnmarshalJSON(data []byte) error {
	pDisk := paramsDisk{}
	err := json.Unmarshal(data, &pDisk)
	if err != nil {
		return err
	}

	*p = Params{
		MinKeys:               pDisk.MinKeys,
		MaxKeys:               pDisk.MaxKeys,
		RekeyThreshold:        pDisk.RekeyThreshold,
		NumRekeys:             pDisk.NumRekeys,
		UnconfirmedRetryRatio: pDisk.UnconfirmedRetryRatio,
	}
	return nil
}

func (p Params) String() string {
	return fmt.Sprintf("SessionParams{ MinKeys: %d, MaxKeys: %d, NumRekeys: %d }",
		p.MinKeys, p.MaxKeys, p.NumRekeys)
}
