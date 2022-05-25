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
func (c *Params) Marshal() ([]byte, error) {
	return json.Marshal(c)
}

// UnmarshalJSON adheres to the json.Unmarshaler interface.
func (c *Params) UnmarshalJSON(data []byte) error {
	p := GetDefaultParams()
	err := json.Unmarshal(data, &p)
	if err != nil {
		return err
	}

	*c = p
	return nil
}

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

func GetDefaultEphemeralParams() Params {
	p := GetDefaultParams()
	p.TriggerName = keyExchangeTriggerEphemeralName
	p.Trigger = catalog.KeyExchangeTriggerEphemeral
	p.ConfirmName = keyExchangeConfirmEphemeralName
	p.Confirm = catalog.KeyExchangeConfirmEphemeral
	p.StoppableName = keyExchangeEphemeralMulti
	return p
}
