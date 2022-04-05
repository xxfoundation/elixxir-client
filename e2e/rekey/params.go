package rekey

import (
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
