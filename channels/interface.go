package channels

import (
	"gitlab.com/elixxir/client/cmix"
	cryptoChannel "gitlab.com/elixxir/crypto/channel"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"math"
	"time"
)

var ValidForever = time.Duration(math.MaxInt64)

type Manager interface {
	SendGeneric(channelID *id.ID, messageType MessageType, msg []byte,
		validUntil time.Duration, params cmix.CMIXParams) (
		cryptoChannel.MessageID, id.Round, ephemeral.Id, error)
	SendAdminGeneric(privKey *rsa.PrivateKey, channelID *id.ID,
		msg []byte, validUntil time.Duration, messageType MessageType,
		params cmix.CMIXParams) (cryptoChannel.MessageID, id.Round, ephemeral.Id,
		error)

	SendMessage(channelID *id.ID, msg string,
		validUntil time.Duration, params cmix.CMIXParams) (
		cryptoChannel.MessageID, id.Round, ephemeral.Id, error)
	SendReply(channelID *id.ID, msg string, replyTo cryptoChannel.MessageID,
		validUntil time.Duration, params cmix.CMIXParams) (
		cryptoChannel.MessageID, id.Round, ephemeral.Id, error)
	SendReaction(channelID *id.ID, msg []byte,
		validUntil time.Duration, params cmix.CMIXParams) (
		cryptoChannel.MessageID, id.Round, ephemeral.Id, error)
}
