package e2e

import (
	"encoding/json"
	"time"

	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/cmix"
)

type Params struct {
	// Tag to use to generate the service.
	ServiceTag string
	// Often, for notifications purposes, all messages except the
	// last should use a silent service. This allows a
	LastServiceTag string

	// The parameters adjust how the code behaves if there are not
	// keys available.  the number of times the code will attempt
	// to get a key to encrypt with
	KeyGetRetryCount uint
	// Delay between attempting to get kets
	KeyGeRetryDelay time.Duration

	//Underlying cmix tags.
	// Note: if critical is true, an alternative critical messages
	// system within e2e will be used which preserves privacy
	CMIX cmix.CMIXParams

	// cMix network params
	Network cmix.Params

	//Authorizes the message to use a key reserved for rekeying. Do not use
	//unless sending a rekey
	Rekey bool
}

func GetDefaultParams() Params {
	return Params{
		ServiceTag:     catalog.Silent,
		LastServiceTag: catalog.E2e,

		KeyGetRetryCount: 10,
		KeyGeRetryDelay:  500 * time.Millisecond,

		CMIX:  cmix.GetDefaultCMIXParams(),
		Rekey: false,
	}
}
func (e Params) Marshal() ([]byte, error) {
	return json.Marshal(e)
}

// GetParameters Obtain default E2E parameters, or override with
// given parameters if set
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
