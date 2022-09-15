///////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"encoding/json"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/channels"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/xxdk"
	cryptoBroadcast "gitlab.com/elixxir/crypto/broadcast"
	cryptoChannel "gitlab.com/elixxir/crypto/channel"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"sync"
	"time"
)

////////////////////////////////////////////////////////////////////////////////
// Singleton Tracker                                                          //
////////////////////////////////////////////////////////////////////////////////

// channelManagerTrackerSingleton is used to track ChannelsManager objects
// so that they can be referenced by ID back over the bindings.
var channelManagerTrackerSingleton = &channelManagerTracker{
	tracked: make(map[int]*ChannelsManager),
	count:   0,
}

// channelManagerTracker is a singleton used to keep track of extant
// ChannelsManager objects, preventing race conditions created by passing it
// over the bindings.
type channelManagerTracker struct {
	tracked map[int]*ChannelsManager
	count   int
	mux     sync.RWMutex
}

// make create a ChannelsManager from an [channels.Manager], assigns it a unique
// ID, and adds it to the channelManagerTracker.
func (cmt *channelManagerTracker) make(c channels.Manager) *ChannelsManager {
	cmt.mux.Lock()
	defer cmt.mux.Unlock()

	chID := cmt.count
	cmt.count++

	cmt.tracked[chID] = &ChannelsManager{
		api: c,
		id:  chID,
	}

	return cmt.tracked[chID]
}

// get an ChannelsManager from the channelManagerTracker given its ID.
func (cmt *channelManagerTracker) get(id int) (*ChannelsManager, error) {
	cmt.mux.RLock()
	defer cmt.mux.RUnlock()

	c, exist := cmt.tracked[id]
	if !exist {
		return nil, errors.Errorf(
			"Cannot get ChannelsManager for ID %d, does not exist", id)
	}

	return c, nil
}

// delete removes a ChannelsManager from the channelManagerTracker.
func (cmt *channelManagerTracker) delete(id int) {
	cmt.mux.Lock()
	defer cmt.mux.Unlock()

	delete(cmt.tracked, id)
}

////////////////////////////////////////////////////////////////////////////////
// Basic Channel API                                                          //
////////////////////////////////////////////////////////////////////////////////

// ChannelsManager is a bindings-layer struct that wraps a [channels.Manager]
// interface.
type ChannelsManager struct {
	api channels.Manager
	id  int
}

// GetID returns the channelManagerTracker ID for the ChannelsManager object.
func (cm *ChannelsManager) GetID() int {
	return cm.id
}

// NewChannelsManager constructs a ChannelsManager.
// FIXME: This is a work in progress and should not be used an event model is
//  implemented in the style of the bindings layer's AuthCallbacks. Remove this
//  note when that has been done.
//
// Parameters:
//  - e2eID - The tracked e2e object ID. This can be retrieved using
//    [E2e.GetID].
//  - udID - The tracked UD object ID. This can be retrieved using
//    [UserDiscovery.GetID].
func NewChannelsManager(e2eID, udID int) (*ChannelsManager, error) {
	// Get user from singleton
	user, err := e2eTrackerSingleton.get(e2eID)
	if err != nil {
		return nil, err
	}

	udMan, err := udTrackerSingleton.get(udID)
	if err != nil {
		return nil, err
	}

	nameService, err := udMan.api.StartChannelNameService()
	if err != nil {
		return nil, err
	}

	// Construct new channels manager
	// TODO: Implement a bindings layer event model, pass that in as a parameter
	//  or the function and pass that into here.
	m := channels.NewManager(user.api.GetStorage().GetKV(), user.api.GetCmix(),
		user.api.GetRng(), nameService, nil)

	// Add channel to singleton and return
	return channelManagerTrackerSingleton.make(m), nil
}

// JoinChannel joins the given channel. It will fail if the channel has already
// been joined.
//
// Parameters:
//  - channelJson - A JSON encoded [ChannelDef].
func (cm *ChannelsManager) JoinChannel(channelJson []byte) error {
	// Unmarshal channel definition
	def := ChannelDef{}
	err := json.Unmarshal(channelJson, &def)
	if err != nil {
		return err
	}

	// Construct ID using the embedded cryptographic information
	channelId, err := cryptoBroadcast.NewChannelID(def.Name, def.Description,
		def.Salt, def.PubKey)
	if err != nil {
		return err
	}

	// Construct public key into object
	rsaPubKey, err := rsa.LoadPublicKeyFromPem(def.PubKey)
	if err != nil {
		return err
	}

	// Construct cryptographic channel object
	channel := &cryptoBroadcast.Channel{
		ReceptionID: channelId,
		Name:        def.Name,
		Description: def.Description,
		Salt:        def.Salt,
		RsaPubKey:   rsaPubKey,
	}

	// Join the channel using the API
	return cm.api.JoinChannel(channel)
}

// GetChannels returns the IDs of all channels that have been joined.
//
// Returns:
//  - []byte - A JSON marshalled list of IDs.
//
// JSON Example:
//  {
//    U4x/lrFkvxuXu59LtHLon1sUhPJSCcnZND6SugndnVID",
//    "15tNdkKbYXoMn58NO6VbDMDWFEyIhTWEGsvgcJsHWAgD"
//  }
func (cm *ChannelsManager) GetChannels() ([]byte, error) {
	channelIds := cm.api.GetChannels()
	return json.Marshal(channelIds)
}

// GetChannelId returns the ID of the channel given the channel's cryptographic
// information.
//
// Parameters:
//  - channelJson - A JSON encoded [ChannelDef]. This may be retrieved from
//    [Channel.Get], for example.
//
// Returns:
//  - []byte - A JSON encoded channel ID ([id.ID]).
//
// JSON Example:
//  "dGVzdAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD"
func (cm *ChannelsManager) GetChannelId(channelJson []byte) ([]byte, error) {
	def := ChannelDef{}
	err := json.Unmarshal(channelJson, &def)
	if err != nil {
		return nil, err
	}

	channelId, err := cryptoBroadcast.NewChannelID(def.Name, def.Description,
		def.Salt, def.PubKey)
	if err != nil {
		return nil, err
	}

	return json.Marshal(channelId)
}

// GetChannel returns the underlying cryptographic structure for a given
// channel.
//
// Parameters:
//  - marshalledChanId - A JSON marshalled channel ID ([id.ID]). This may be
//    retrieved using ChannelsManager.GetChannelId.
//
// Returns:
//  - []byte - A JSON marshalled ChannelDef.
func (cm *ChannelsManager) GetChannel(marshalledChanId []byte) ([]byte, error) {
	// Unmarshal ID
	chanId, err := id.Unmarshal(marshalledChanId)
	if err != nil {
		return nil, err
	}

	// Retrieve channel from manager
	def, err := cm.api.GetChannel(chanId)
	if err != nil {
		return nil, err
	}

	// Marshal channel
	return json.Marshal(&ChannelDef{
		Name:        def.Name,
		Description: def.Description,
		Salt:        def.Salt,
		PubKey:      rsa.CreatePublicKeyPem(def.RsaPubKey),
	})
}

// LeaveChannel leaves the given channel. It will return an error if the
// channel was not previously joined.
//
// Parameters:
//  - marshalledChanId - A JSON marshalled channel ID ([id.ID]). This may be
//    retrieved using ChannelsManager.GetChannelId.
func (cm *ChannelsManager) LeaveChannel(marshalledChanId []byte) error {
	// Unmarshal channel ID
	channelId, err := id.Unmarshal(marshalledChanId)
	if err != nil {
		return err
	}

	// Leave the channel
	return cm.api.LeaveChannel(channelId)
}

// ReplayChannel replays all messages from the channel within the network's
// memory (~3 weeks) over the event model.
//
// Parameters:
//  - marshalledChanId - A JSON marshalled channel ID ([id.ID]). This may be
//    retrieved using ChannelsManager.GetChannelId.
func (cm *ChannelsManager) ReplayChannel(marshalledChanId []byte) error {

	// Unmarshal channel ID
	chanId, err := id.Unmarshal(marshalledChanId)
	if err != nil {
		return err
	}

	// Replay channel
	return cm.api.ReplayChannel(chanId)
}

////////////////////////////////////////////////////////////////////////////////
// Channel Sending Methods & Reports                                          //
////////////////////////////////////////////////////////////////////////////////

// ChannelSendReport is the bindings' representation of the return values of
// ChannelsManager's Send operations.
//
// JSON Example:
//  {
//    "MessageId": "0kitNxoFdsF4q1VMSI/xPzfCnGB2l+ln2+7CTHjHbJw=",
//    "Rounds":[1,5,9],
//    "EphId": 0
//  }
type ChannelSendReport struct {
	MessageId []byte
	RoundsList
	EphId int64
}

// SendGeneric is used to send a raw message over a channel. In general, it
// should be wrapped in a function that defines the wire protocol. If the final
// message, before being sent over the wire, is too long, this will return an
// error. Due to the underlying encoding using compression, it isn't possible to
// define the largest payload that can be sent, but it will always be possible
// to send a payload of 802 bytes at minimum. The meaning of validUntil depends
// on the use case.
//
// Parameters:
//  - marshalledChanId - A JSON marshalled channel ID ([id.ID]). This may be
//    retrieved using ChannelsManager.GetChannelId.
//  - messageType - The message type of the message. This will be a valid
//    [channels.MessageType].
//  - message - The contents of the message. This need not be of data type
//    string, as the message could be a specified format that the channel may
//    recognize.
//  - leaseTimeMS - The lease of the message. This will be how long the message
//    is valid until, in milliseconds. As per the channels.Manager
//    documentation, this has different meanings depending on the use case.
//    These use cases may be generic enough that they will not be enumerated
//    here.
//  - cmixParamsJSON - A JSON marshalled [xxdk.CMIXParams]. This may be empty,
//    and GetDefaultCMixParams will be used internally.
//
// Returns:
//  - []byte - A JSON marshalled ChannelSendReport.
func (cm *ChannelsManager) SendGeneric(marshalledChanId []byte,
	messageType int, message []byte, leaseTimeMS int64,
	cmixParamsJSON []byte) ([]byte, error) {

	// Unmarshal channel ID and parameters
	chanId, params, err := parseChannelsParameters(
		marshalledChanId, cmixParamsJSON)
	if err != nil {
		return nil, err
	}

	// Send message
	chanMsgId, rnd, ephId, err := cm.api.SendGeneric(chanId,
		channels.MessageType(messageType), message,
		time.Duration(leaseTimeMS), params.CMIX)
	if err != nil {
		return nil, err
	}

	// Construct send report
	return constructChannelSendReport(chanMsgId, rnd.ID, ephId)
}

// SendAdminGeneric is used to send a raw message over a channel encrypted with
// admin keys, identifying it as sent by the admin. In general, it should be
// wrapped in a function that defines the wire protocol. If the final message,
// before being sent over the wire, is too long, this will return an error. The
// message must be at most 510 bytes long.
//
// Parameters:
//  - adminPrivateKey - The PEM-encoded admin RSA private key.
//  - marshalledChanId - A JSON marshalled channel ID ([id.ID]). This may be
//    retrieved using ChannelsManager.GetChannelId.
//  - messageType - The message type of the message. This will be a valid
//    [channels.MessageType].
//  - message - The contents of the message. The message should be at most 510
//    bytes. This need not be of data type string, as the message could be a
//    specified format that the channel may recognize.
//  - leaseTimeMS - The lease of the message. This will be how long the message
//    is valid until, in milliseconds. As per the channels.Manager
//    documentation, this has different meanings depending on the use case.
//    These use cases may be generic enough that they will not be enumerated
//    here.
//  - cmixParamsJSON - A JSON marshalled [xxdk.CMIXParams]. This may be empty,
//    and GetDefaultCMixParams will be used internally.
//
// Returns:
//  - []byte - A JSON marshalled ChannelSendReport.
func (cm *ChannelsManager) SendAdminGeneric(adminPrivateKey,
	marshalledChanId []byte,
	messageType int, message []byte, leaseTimeMS int64,
	cmixParamsJSON []byte) ([]byte, error) {

	// Load private key from file
	rsaPrivKey, err := rsa.LoadPrivateKeyFromPem(adminPrivateKey)
	if err != nil {
		return nil, err
	}

	// Unmarshal channel ID and parameters
	chanId, params, err := parseChannelsParameters(
		marshalledChanId, cmixParamsJSON)
	if err != nil {
		return nil, err
	}

	// Send admin message
	chanMsgId, rnd, ephId, err := cm.api.SendAdminGeneric(rsaPrivKey,
		chanId, channels.MessageType(messageType), message,
		time.Duration(leaseTimeMS), params.CMIX)

	// Construct send report
	return constructChannelSendReport(chanMsgId, rnd.ID, ephId)
}

// SendMessage is used to send a formatted message over a channel.
// Due to the underlying encoding using compression, it isn't possible to define
// the largest payload that can be sent, but it will always be possible to send
// a payload of 798 bytes at minimum.
//
// The message will auto delete validUntil after the round it is sent in,
// lasting forever if [channels.ValidForever] is used.
//
// Parameters:
//  - marshalledChanId - A JSON marshalled channel ID ([id.ID]). This may be
//    retrieved using ChannelsManager.GetChannelId.
//  - message - The contents of the message. The message should be at most 510
//    bytes. This is expected to be Unicode, and thus a string data type is
//    expected
//  - leaseTimeMS - The lease of the message. This will be how long the message
//    is valid until, in milliseconds. As per the channels.Manager
//    documentation, this has different meanings depending on the use case.
//    These use cases may be generic enough that they will not be enumerated
//    here.
//  - cmixParamsJSON - A JSON marshalled [xxdk.CMIXParams]. This may be
//    empty, and GetDefaultCMixParams will be used internally.
//
// Returns:
//  - []byte - A JSON marshalled ChannelSendReport
func (cm *ChannelsManager) SendMessage(marshalledChanId []byte,
	message string, leaseTimeMS int64, cmixParamsJSON []byte) ([]byte, error) {

	// Unmarshal channel ID and parameters
	chanId, params, err := parseChannelsParameters(
		marshalledChanId, cmixParamsJSON)
	if err != nil {
		return nil, err
	}

	// Send message
	chanMsgId, rnd, ephId, err := cm.api.SendMessage(chanId, message,
		time.Duration(leaseTimeMS), params.CMIX)
	if err != nil {
		return nil, err
	}

	// Construct send report
	return constructChannelSendReport(chanMsgId, rnd.ID, ephId)
}

// SendReply is used to send a formatted message over a channel.
// Due to the underlying encoding using compression, it isn't possible to define
// the largest payload that can be sent, but it will always be possible to send
// a payload of 766 bytes at minimum.
//
// If the message ID the reply is sent to does not exist, then the other side
// will post the message as a normal message and not a reply.
// The message will auto delete validUntil after the round it is sent in,
// lasting forever if ValidForever is used.
//
// Parameters:
//  - marshalledChanId - A JSON marshalled channel ID ([id.ID]). This may be
//    retrieved using ChannelsManager.GetChannelId.
//  - message - The contents of the message. The message should be at most 510
//    bytes. This is expected to be Unicode, and thus a string data type is
//    expected.
//  - messageToReactTo - The marshalled [channel.MessageID] of the message you
//    wish to reply to. This may be found in the ChannelSendReport if replying
//    to your own. Alternatively, if reacting to another user's message, you may
//    retrieve it via the ChannelMessageReceptionCallback registered using
//    RegisterReceiveHandler.
//  - leaseTimeMS - The lease of the message. This will be how long the message
//    is valid until, in milliseconds. As per the channels.Manager
//    documentation, this has different meanings depending on the use case.
//    These use cases may be generic enough that they will not be enumerated
//    here.
//  - cmixParamsJSON - A JSON marshalled [xxdk.CMIXParams]. This may be empty,
//    and GetDefaultCMixParams will be used internally.
//
// Returns:
//  - []byte - A JSON marshalled ChannelSendReport
func (cm *ChannelsManager) SendReply(marshalledChanId []byte,
	message string, messageToReactTo []byte, leaseTimeMS int64,
	cmixParamsJSON []byte) ([]byte, error) {

	// Unmarshal channel ID and parameters
	chanId, params, err := parseChannelsParameters(
		marshalledChanId, cmixParamsJSON)
	if err != nil {
		return nil, err
	}

	// Unmarshal message ID
	msgId := cryptoChannel.MessageID{}
	copy(msgId[:], messageToReactTo)

	// Send Reply
	chanMsgId, rnd, ephId, err := cm.api.SendReply(chanId, message,
		msgId, time.Duration(leaseTimeMS), params.CMIX)
	if err != nil {
		return nil, err
	}

	// Construct send report
	return constructChannelSendReport(chanMsgId, rnd.ID, ephId)
}

// SendReaction is used to send a reaction to a message over a channel.
// The reaction must be a single emoji with no other characters, and will
// be rejected otherwise.
// Users will drop the reaction if they do not recognize the reactTo message.
//
// Parameters:
//  - marshalledChanId - A JSON marshalled channel ID ([id.ID]). This may be
//    retrieved using ChannelsManager.GetChannelId.
//  - reaction - The user's reaction. This should be a single emoji with no
//    other characters. As such, a Unicode string is expected.
//  - messageToReactTo - The marshalled [channel.MessageID] of the message you
//    wish to reply to. This may be found in the ChannelSendReport if replying
//    to your own. Alternatively, if reacting to another user's message, you may
//    retrieve it via the ChannelMessageReceptionCallback registered using
//    RegisterReceiveHandler.
//  - cmixParamsJSON - A JSON marshalled [xxdk.CMIXParams]. This may be empty,
//  and GetDefaultCMixParams will be used internally.
//
// Returns:
//  - []byte - A JSON marshalled ChannelSendReport.
func (cm *ChannelsManager) SendReaction(marshalledChanId []byte,
	reaction string, messageToReactTo []byte,
	cmixParamsJSON []byte) ([]byte, error) {

	// Unmarshal channel ID and parameters
	chanId, params, err := parseChannelsParameters(
		marshalledChanId, cmixParamsJSON)
	if err != nil {
		return nil, err
	}

	// Unmarshal message ID
	msgId := cryptoChannel.MessageID{}
	copy(msgId[:], messageToReactTo)

	// Send reaction
	chanMsgId, rnd, ephId, err := cm.api.SendReaction(chanId,
		reaction, msgId, params.CMIX)
	if err != nil {
		return nil, err
	}

	// Construct send report
	return constructChannelSendReport(chanMsgId, rnd.ID, ephId)
}

// parseChannelsParameters is a helper function for the Send functions. It
// parses the channel ID and the passed in parameters into their respective
// objects. These objects are passed into the API via the internal send
// functions.
func parseChannelsParameters(marshalledChanId, cmixParamsJSON []byte) (
	*id.ID, xxdk.CMIXParams, error) {
	// Unmarshal channel ID
	chanId, err := id.Unmarshal(marshalledChanId)
	if err != nil {
		return nil, xxdk.CMIXParams{}, err
	}

	// Unmarshal cmix params
	params, err := parseCMixParams(cmixParamsJSON)
	if err != nil {
		return nil, xxdk.CMIXParams{}, err
	}

	return chanId, params, nil
}

// constructChannelSendReport is a helper function which returns a JSON
// marshalled ChannelSendReport.
func constructChannelSendReport(channelMessageId cryptoChannel.MessageID,
	roundId id.Round, ephId ephemeral.Id) ([]byte, error) {
	// Construct send report
	chanSendReport := ChannelSendReport{
		MessageId:  channelMessageId.Bytes(),
		RoundsList: makeRoundsList(roundId),
		EphId:      ephId.Int64(),
	}

	// Marshal send report
	return json.Marshal(chanSendReport)
}

////////////////////////////////////////////////////////////////////////////////
// Channel Receiving Logic and Callback Registration                          //
////////////////////////////////////////////////////////////////////////////////

// ReceivedChannelMessageReport is a report structure returned via the
// ChannelMessageReceptionCallback. This report gives the context for the
// channel the message was sent to and the message itself. This is returned via
// the callback as JSON marshalled bytes.
//
// JSON Example:
//  {
//    "ChannelId": "AQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
//    "MessageId": "3S6DiVjWH9mLmjy1oaam/3x45bJQzOW6u2KgeUn59wA=",
//    "ReplyTo":"cxMyGUFJ+Ff1Xp2X+XkIpOnNAQEZmv8SNP5eYH4tCik=",
//    "MessageType": 42,
//    "SenderUsername": "hunter2",
//    "Content": "YmFuX2JhZFVTZXI=",
//    "Timestamp": 1662502150335283000,
//    "Lease": 25,
//    "Rounds": [ 1, 4, 9],
//  }
type ReceivedChannelMessageReport struct {
	ChannelId      []byte
	MessageId      []byte
	MessageType    int
	SenderUsername string
	Content        []byte
	Timestamp      int64
	Lease          int64
	RoundsList
}

// ChannelMessageReceptionCallback is the callback that returns the context for
// a channel message via the Callback.
type ChannelMessageReceptionCallback interface {
	Callback(receivedChannelMessageReport []byte, err error)
}

// RegisterReceiveHandler is used to register handlers for non-default message
// types. They can be processed by modules. It is important that such modules
// sync up with the event model implementation.
//
// There can only be one handler per [channels.MessageType], and this will
// return an error on any re-registration.
//
// Parameters:
//  - messageType - represents the [channels.MessageType] which will have a
//    registered listener.
//  - listenerCb - the callback which will be executed when a channel message
//    of messageType is received.
func (cm *ChannelsManager) RegisterReceiveHandler(messageType int,
	listenerCb ChannelMessageReceptionCallback) error {

	// Wrap callback around backend interface
	cb := channels.MessageTypeReceiveMessage(
		func(channelID *id.ID,
			messageID cryptoChannel.MessageID, messageType channels.MessageType,
			senderUsername string, content []byte, timestamp time.Time,
			lease time.Duration, round rounds.Round, status channels.SentStatus) {

			rcm := ReceivedChannelMessageReport{
				ChannelId:      channelID.Marshal(),
				MessageId:      messageID.Bytes(),
				MessageType:    int(messageType),
				SenderUsername: senderUsername,
				Content:        content,
				Timestamp:      timestamp.UnixNano(),
				Lease:          int64(lease),
				RoundsList:     makeRoundsList(round.ID),
			}

			listenerCb.Callback(json.Marshal(rcm))
		})

	// Register handler
	return cm.api.RegisterReceiveHandler(channels.MessageType(messageType), cb)
}
