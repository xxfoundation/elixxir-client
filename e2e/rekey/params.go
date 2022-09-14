////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package rekey

import (
	"encoding/json"
	"gitlab.com/elixxir/client/catalog"
	"time"
)

const keyExchangeTriggerName = "KeyExchangeTrigger"
const keyExchangeConfirmName = "KeyExchangeConfirm"
const keyExchangeMulti = "KeyExchange"

const keyExchangeTriggerEphemeralName = "KeyExchangeTriggerEphemeral"
const keyExchangeConfirmEphemeralName = "KeyExchangeConfirmEphemeral"
const keyExchangeEphemeralMulti = "KeyExchangeEphemeral"

type Params struct {
	RoundTimeout  time.Duration
	TriggerName   string
	Trigger       catalog.MessageType
	ConfirmName   string
	Confirm       catalog.MessageType
	StoppableName string
}

// paramsDisk will be the marshal-able and umarshal-able object.
type paramsDisk struct {
	RoundTimeout  time.Duration
	TriggerName   string
	Trigger       catalog.MessageType
	ConfirmName   string
	Confirm       catalog.MessageType
	StoppableName string
}

// GetDefaultParams returns a default set of Params.
func GetDefaultParams() Params {
	return Params{
		RoundTimeout:  time.Minute,
		TriggerName:   keyExchangeTriggerName,
		Trigger:       catalog.KeyExchangeTrigger,
		ConfirmName:   keyExchangeConfirmName,
		Confirm:       catalog.KeyExchangeConfirm,
		StoppableName: keyExchangeMulti,
	}
}

// GetDefaultEphemeralParams returns a default set of Params for
// ephemeral re-keying.
func GetDefaultEphemeralParams() Params {
	p := GetDefaultParams()
	p.TriggerName = keyExchangeTriggerEphemeralName
	p.Trigger = catalog.KeyExchangeTriggerEphemeral
	p.ConfirmName = keyExchangeConfirmEphemeralName
	p.Confirm = catalog.KeyExchangeConfirmEphemeral
	p.StoppableName = keyExchangeEphemeralMulti
	return p
}

// GetParameters returns the default network parameters, or override with given
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
		RoundTimeout:  p.RoundTimeout,
		TriggerName:   p.TriggerName,
		Trigger:       p.Trigger,
		ConfirmName:   p.ConfirmName,
		Confirm:       p.Confirm,
		StoppableName: p.StoppableName,
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
		RoundTimeout:  pDisk.RoundTimeout,
		TriggerName:   pDisk.TriggerName,
		Trigger:       pDisk.Trigger,
		ConfirmName:   pDisk.ConfirmName,
		Confirm:       pDisk.Confirm,
		StoppableName: pDisk.StoppableName,
	}

	return nil
}
