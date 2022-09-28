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
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/channels"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/xxdk"
	cryptoBroadcast "gitlab.com/elixxir/crypto/broadcast"
	cryptoChannel "gitlab.com/elixxir/crypto/channel"
	"gitlab.com/elixxir/crypto/rsa"
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

// GenerateChannelIdentity creates a new private channel identity
// ([channel.PrivateIdentity]). The public component can be retrieved as JSON
// via [GetPublicChannelIdentityFromPrivate].
//
// Parameters:
//  - cmixID - The tracked cmix object ID. This can be retrieved using
//    [Cmix.GetID].
//
// Returns:
//  - JSON of [channel.PrivateIdentity].
func GenerateChannelIdentity(cmixID int) ([]byte, error) {
	// Get user from singleton
	user, err := cmixTrackerSingleton.get(cmixID)
	if err != nil {
		return nil, err
	}

	rng := user.api.GetRng().GetStream()
	defer rng.Close()
	pi, err := cryptoChannel.GenerateIdentity(rng)
	if err != nil {
		return nil, err
	}
	return pi.Marshal(), nil
}

// GetPublicChannelIdentity constructs a public identity ([channel.Identity])
// from a bytes version and returns it JSON marshaled.
//
// Parameters:
//  - marshaledPublic - Bytes of the public identity ([channel.Identity]).
//
// Returns:
//  - JSON of the constructed [channel.Identity].
func GetPublicChannelIdentity(marshaledPublic []byte) ([]byte, error) {
	i, err := cryptoChannel.UnmarshalIdentity(marshaledPublic)
	if err != nil {
		return nil, err
	}
	return json.Marshal(&i)
}

// GetPublicChannelIdentityFromPrivate returns the public identity
// ([channel.Identity]) contained in the given private identity
// ([channel.PrivateIdentity]).
//
// Parameters:
//  - marshaledPrivate - Bytes of the private identity
//    (channel.PrivateIdentity]).
//
// Returns:
//  - JSON of the public [channel.Identity].
func GetPublicChannelIdentityFromPrivate(marshaledPrivate []byte) ([]byte, error) {
	pi, err := cryptoChannel.UnmarshalPrivateIdentity(marshaledPrivate)
	if err != nil {
		return nil, err
	}
	return json.Marshal(&pi.Identity)
}

// NewChannelsManagerGoEventModel creates a new [ChannelsManager] from a new
// private identity ([channel.PrivateIdentity]). This is not compatible with
// GoMobile Bindings because it receives the go event model.
//
// This is for creating a manager for an identity for the first time. For
// generating a new one channel identity, use [GenerateChannelIdentity]. To
// reload this channel manager, use [LoadChannelsManagerGoEventModel], passing
// in the storage tag retrieved by [ChannelsManager.GetStorageTag].
//
// Parameters:
//  - cmixID - The tracked Cmix object ID. This can be retrieved using
//    [Cmix.GetID].
//  - privateIdentity - Bytes of a private identity ([channel.PrivateIdentity])
//    that is generated by [GenerateChannelIdentity].
//  - goEvent - A function that initialises and returns the event model that is
//    not compatible with GoMobile bindings.
func NewChannelsManagerGoEventModel(cmixID int, privateIdentity []byte,
	goEventBuilder channels.EventModelBuilder) (*ChannelsManager, error) {
	pi, err := cryptoChannel.UnmarshalPrivateIdentity(privateIdentity)
	if err != nil {
		return nil, err
	}

	// Get user from singleton
	user, err := cmixTrackerSingleton.get(cmixID)
	if err != nil {
		return nil, err
	}

	// Construct new channels manager
	m, err := channels.NewManager(pi, user.api.GetStorage().GetKV(),
		user.api.GetCmix(), user.api.GetRng(), goEventBuilder)
	if err != nil {
		return nil, err
	}

	// Add channel to singleton and return
	return channelManagerTrackerSingleton.make(m), nil
}

// LoadChannelsManagerGoEventModel loads an existing ChannelsManager. This is not
// compatible with GoMobile Bindings because it receives the go event model.
// This is for creating a manager for an identity for the first time.
// The channel manager should have first been created with
// NewChannelsManagerGoEventModel and then the storage tag can be retrieved
// with ChannelsManager.GetStorageTag
//
// Parameters:
//  - cmixID - The tracked cmix object ID. This can be retrieved using
//    [Cmix.GetID].
//  - storageTag - retrieved with ChannelsManager.GetStorageTag
//  - goEvent - A function that initialises and returns the event model that is
//    not compatible with GoMobile bindings.
func LoadChannelsManagerGoEventModel(cmixID int, storageTag string,
	goEventBuilder channels.EventModelBuilder) (*ChannelsManager, error) {

	// Get user from singleton
	user, err := cmixTrackerSingleton.get(cmixID)
	if err != nil {
		return nil, err
	}

	// Construct new channels manager
	m, err := channels.LoadManager(storageTag, user.api.GetStorage().GetKV(),
		user.api.GetCmix(), user.api.GetRng(), goEventBuilder)
	if err != nil {
		return nil, err
	}

	// Add channel to singleton and return
	return channelManagerTrackerSingleton.make(m), nil
}

// NewChannelsManager creates a new [ChannelsManager] from a new private
// identity ([channel.PrivateIdentity]).
//
// This is for creating a manager for an identity for the first time. For
// generating a new one channel identity, use [GenerateChannelIdentity]. To
// reload this channel manager, use [LoadChannelsManager], passing in the
// storage tag retrieved by [ChannelsManager.GetStorageTag].
//
// Parameters:
//  - cmixID - The tracked Cmix object ID. This can be retrieved using
//    [Cmix.GetID].
//  - privateIdentity - Bytes of a private identity ([channel.PrivateIdentity])
//    that is generated by [GenerateChannelIdentity].
//  - event -  An interface that contains a function that initialises and returns
//    the event model that is bindings-compatible.
func NewChannelsManager(cmixID int, privateIdentity []byte,
	eventBuilder EventModelBuilder) (*ChannelsManager, error) {
	pi, err := cryptoChannel.UnmarshalPrivateIdentity(privateIdentity)
	if err != nil {
		return nil, err
	}

	// Get user from singleton
	user, err := cmixTrackerSingleton.get(cmixID)
	if err != nil {
		return nil, err
	}

	eb := func(path string) channels.EventModel {
		return NewEventModel(eventBuilder.Build(path))
	}

	// Construct new channels manager
	m, err := channels.NewManager(pi, user.api.GetStorage().GetKV(),
		user.api.GetCmix(), user.api.GetRng(), eb)
	if err != nil {
		return nil, err
	}

	// Add channel to singleton and return
	return channelManagerTrackerSingleton.make(m), nil
}

// LoadChannelsManager loads an existing [ChannelsManager].
//
// This is for loading a manager for an identity that has already been created.
// The channel manager should have previously been created with
// [NewChannelsManager] and the storage is retrievable with
// [ChannelsManager.GetStorageTag].
//
// Parameters:
//  - cmixID - The tracked cmix object ID. This can be retrieved using
//    [Cmix.GetID].
//  - storageTag - The storage tag associated with the previously created
//    channel manager and retrieved with [ChannelsManager.GetStorageTag].
//  - event - An interface that contains a function that initialises and returns
//    the event model that is bindings-compatible.
func LoadChannelsManager(cmixID int, storageTag string,
	eventBuilder EventModelBuilder) (*ChannelsManager, error) {

	// Get user from singleton
	user, err := cmixTrackerSingleton.get(cmixID)
	if err != nil {
		return nil, err
	}

	eb := func(path string) channels.EventModel {
		return NewEventModel(eventBuilder.Build(path))
	}

	// Construct new channels manager
	m, err := channels.LoadManager(storageTag, user.api.GetStorage().GetKV(),
		user.api.GetCmix(), user.api.GetRng(), eb)
	if err != nil {
		return nil, err
	}

	// Add channel to singleton and return
	return channelManagerTrackerSingleton.make(m), nil
}

type ChannelGeneration struct {
	Channel    string
	PrivateKey string
}

// GenerateChannel is used to create a channel a new channel of which you are
// the admin. It is only for making new channels, not joining existing ones.
//
// It returns a pretty print of the channel and the private key.
//
// The name cannot be longer that __ characters. The description cannot be
// longer than __ and can only use ______ characters.
//
// Parameters:
//  - cmixID - The tracked cmix object ID. This can be retrieved using
//    [Cmix.GetID].
//  - name - The name of the new channel. The name cannot be longer than __
//    characters and must contain only _____ characters. It cannot be changed
//    once a channel is created.
//  - description - The description of a channel. The description cannot be
//    longer than __ characters and must contain only _____ characters. It
//    cannot be changed once a channel is created.
//
// Returns:
//  - []byte - ChannelGeneration describes a generated channel. It contains both
//    the public channel info and the private key for the channel in PEM format.
//    fixme: document json
func GenerateChannel(cmixID int, name, description string) ([]byte, error) {
	// Get cmix from singleton so its rng can be used
	cmix, err := cmixTrackerSingleton.get(cmixID)
	if err != nil {
		return nil, err
	}

	stream := cmix.api.GetRng().GetStream()
	defer stream.Close()
	c, pk, err := cryptoBroadcast.NewChannel(
		name, description, cmix.api.GetCmix().GetMaxMessageLength(), stream)
	if err != nil {
		return nil, err
	}

	gen := ChannelGeneration{
		Channel:    c.PrettyPrint(),
		PrivateKey: string(pk.MarshalPem()),
	}

	return json.Marshal(&gen)
}

type ChannelInfo struct {
	Name        string
	Description string
	ChannelID   string
}

// GetChannelInfo returns the info about a channel from its public description.
//
// Parameters:
//  - prettyPrint - The pretty print of the channel.
//
// The pretty print will be of the format:
//  <XXChannel-v1:Test Channel,description:This is a test channel,secrets:pn0kIs6P1pHvAe7u8kUyf33GYVKmkoCX9LhCtvKJZQI=,3A5eB5pzSHyxN09w1kOVrTIEr5UyBbzmmd9Ga5Dx0XA=,0,0,/zChIlLr2p3Vsm2X4+3TiFapoapaTi8EJIisJSqwfGc=>
//
// Returns:
//  - []byte - ChannelInfo describes all relevant channel info.
//    fixme: document json
func GetChannelInfo(prettyPrint string) ([]byte, error) {
	_, bytes, err := getChannelInfo(prettyPrint)
	return bytes, err
}

func getChannelInfo(prettyPrint string) (*cryptoBroadcast.Channel, []byte, error) {
	c, err := cryptoBroadcast.NewChannelFromPrettyPrint(prettyPrint)
	if err != nil {
		return nil, nil, err
	}
	ci := &ChannelInfo{
		Name:        c.Name,
		Description: c.Description,
		ChannelID:   c.ReceptionID.String(),
	}
	bytes, err := json.Marshal(ci)
	if err != nil {
		return nil, nil, err
	}
	return c, bytes, nil
}

// JoinChannel joins the given channel. It will fail if the channel has already
// been joined.
//
// Parameters:
//  - channelPretty - A portable channel string. Should be received from
//    another user or generated via GenerateChannel.
//
// The pretty print will be of the format:
//  <XXChannel-v1:Test Channel,description:This is a test channel,secrets:pn0kIs6P1pHvAe7u8kUyf33GYVKmkoCX9LhCtvKJZQI=,3A5eB5pzSHyxN09w1kOVrTIEr5UyBbzmmd9Ga5Dx0XA=,0,0,/zChIlLr2p3Vsm2X4+3TiFapoapaTi8EJIisJSqwfGc=>"
//
// Returns:
//  - []byte - ChannelInfo describes all relevant channel info.
//    fixme: document json
func (cm *ChannelsManager) JoinChannel(channelPretty string) ([]byte, error) {
	c, info, err := getChannelInfo(channelPretty)
	if err != nil {
		return nil, err
	}

	// Join the channel using the API
	err = cm.api.JoinChannel(c)

	return info, err
}

// GetChannels returns the IDs of all channels that have been joined.
//
// Returns:
//  - []byte - A JSON marshalled list of IDs.
//
// JSON Example:
//  {
//    "U4x/lrFkvxuXu59LtHLon1sUhPJSCcnZND6SugndnVID",
//    "15tNdkKbYXoMn58NO6VbDMDWFEyIhTWEGsvgcJsHWAgD"
//  }
func (cm *ChannelsManager) GetChannels() ([]byte, error) {
	channelIds := cm.api.GetChannels()
	return json.Marshal(channelIds)
}

// LeaveChannel leaves the given channel. It will return an error if the
// channel was not previously joined.
//
// Parameters:
//  - marshalledChanId - A JSON marshalled channel ID ([id.ID]).
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
//  - marshalledChanId - A JSON marshalled channel ID ([id.ID]).
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
//  - marshalledChanId - A JSON marshalled channel ID ([id.ID]).
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
//  - marshalledChanId - A JSON marshalled channel ID ([id.ID]).
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
	rsaPrivKey, err := rsa.GetScheme().UnmarshalPrivateKeyPEM(adminPrivateKey)
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
//  - marshalledChanId - A JSON marshalled channel ID ([id.ID]).
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

// SendReply is used to send a formatted message over a channel. Due to the
// underlying encoding using compression, it isn't possible to define the
// largest payload that can be sent, but it will always be possible to send a
// payload of 766 bytes at minimum.
//
// If the message ID the reply is sent to is nonexistent, the other side will
// post the message as a normal message and not a reply. The message will auto
// delete validUntil after the round it is sent in, lasting forever if
// [channels.ValidForever] is used.
//
// Parameters:
//  - marshalledChanId - A JSON marshalled channel ID ([id.ID]).
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
//  - marshalledChanId - A JSON marshalled channel ID ([id.ID]).
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

// GetIdentity returns the marshaled public identity ([channel.Identity]) that
// the channel is using.
func (cm *ChannelsManager) GetIdentity() ([]byte, error) {
	i := cm.api.GetIdentity()
	return json.Marshal(&i)
}

// GetStorageTag returns the storage tag needed to reload the manager.
func (cm *ChannelsManager) GetStorageTag() string {
	return cm.api.GetStorageTag()
}

// SetNickname sets the nickname for a given channel. The nickname must be valid
// according to [IsNicknameValid].
func (cm *ChannelsManager) SetNickname(newNick string, ch []byte) error {
	chid, err := id.Unmarshal(ch)
	if err != nil {
		return err
	}
	return cm.api.SetNickname(newNick, chid)
}

// DeleteNickname deletes the nickname for a given channel.
func (cm *ChannelsManager) DeleteNickname(ch []byte) error {
	chid, err := id.Unmarshal(ch)
	if err != nil {
		return err
	}
	return cm.api.DeleteNickname(chid)
}

// GetNickname returns the nickname set for a given channel. Returns an error if
// there is no nickname set.
func (cm *ChannelsManager) GetNickname(ch []byte) (string, error) {
	chid, err := id.Unmarshal(ch)
	if err != nil {
		return "", err
	}
	nick, exists := cm.api.GetNickname(chid)
	if !exists {
		return "", errors.New("no nickname found for the given channel")
	}

	return nick, nil
}

// IsNicknameValid checks if a nickname is valid.
//
// Rules:
//  1. A nickname must not be longer than 24 characters.
//  2. A nickname must not be shorter than 1 character.
func IsNicknameValid(nick string) error {
	return channels.IsNicknameValid(nick)
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
	ChannelId   []byte
	MessageId   []byte
	MessageType int
	Nickname    string
	Identity    []byte
	Content     []byte
	Timestamp   int64
	Lease       int64
	RoundsList
}

// ChannelMessageReceptionCallback is the callback that returns the context for
// a channel message via the Callback.
// It must return a unique UUID for the message by which it can be referenced
// later
type ChannelMessageReceptionCallback interface {
	Callback(receivedChannelMessageReport []byte, err error) int
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
			nickname string, content []byte, identity cryptoChannel.Identity,
			timestamp time.Time, lease time.Duration, round rounds.Round,
			status channels.SentStatus) uint64 {

			idBytes, err := json.Marshal(&identity)
			if err != nil {
				jww.WARN.Printf("failed to marshal identity object: %+v", err)
				return 0
			}

			rcm := ReceivedChannelMessageReport{
				ChannelId:   channelID.Marshal(),
				MessageId:   messageID.Bytes(),
				MessageType: int(messageType),
				Nickname:    nickname,
				Identity:    idBytes,
				Content:     content,
				Timestamp:   timestamp.UnixNano(),
				Lease:       int64(lease),
				RoundsList:  makeRoundsList(round.ID),
			}

			return uint64(listenerCb.Callback(json.Marshal(rcm)))
		})

	// Register handler
	return cm.api.RegisterReceiveHandler(channels.MessageType(messageType), cb)
}

////////////////////////////////////////////////////////////////////////////////
// Event Model Logic                                                          //
////////////////////////////////////////////////////////////////////////////////

// EventModelBuilder builds an event model
type EventModelBuilder interface {
	Build(path string) EventModel
}

// EventModel is an interface which an external party which uses the channels
// system passed an object which adheres to in order to get events on the
// channel.
type EventModel interface {
	// JoinChannel is called whenever a channel is joined locally.
	//
	// Parameters:
	//  - channel - Returns the pretty print representation of a channel.
	JoinChannel(channel string)

	// LeaveChannel is called whenever a channel is left locally.
	//
	// Parameters:
	//  - ChannelId - The marshalled channel [id.ID].
	LeaveChannel(channelID []byte)

	// ReceiveMessage is called whenever a message is received on a given
	// channel. It may be called multiple times on the same message. It is
	// incumbent on the user of the API to filter such called by message ID.
	//
	// Parameters:
	//  - channelID - The marshalled channel [id.ID].
	//  - messageID - The bytes of the [channel.MessageID] of the received
	//    message.
	//  - nickname - The nickname of the sender of the message.
	//  - text - The content of the message.
	//  - timestamp - Time the message was received; represented as nanoseconds
	//    since unix epoch.
	//  - identity - the json of the identity of the sender
	//  - lease - The number of nanoseconds that the message is valid for.
	//  - roundId - The ID of the round that the message was received on.
	//  - status - the [channels.SentStatus] of the message.
	//
	// Statuses will be enumerated as such:
	//  Sent      =  0
	//  Delivered =  1
	//  Failed    =  2
	//
	// Returns a non-negative unique UUID for the message that it can be
	// referenced by later with [EventModel.UpdateSentStatus].
	ReceiveMessage(channelID, messageID []byte, nickname, text string,
		identity []byte, timestamp, lease, roundId, status int64) int64

	// ReceiveReply is called whenever a message is received that is a reply on
	// a given channel. It may be called multiple times on the same message. It
	// is incumbent on the user of the API to filter such called by message ID.
	//
	// Messages may arrive our of order, so a reply in theory can arrive before
	// the initial message. As a result, it may be important to buffer replies.
	//
	// Parameters:
	//  - channelID - The marshalled channel [id.ID].
	//  - messageID - The bytes of the [channel.MessageID] of the received
	//    message.
	//  - reactionTo - The [channel.MessageID] for the message that received a
	//    reply.
	//  - nickname - The nickname of the sender of the message.
	//  - text - The content of the message.
	//  - identity - the json marshaled identity of the sender
	//  - timestamp - Time the message was received; represented as nanoseconds
	//    since unix epoch.
	//  - lease - The number of nanoseconds that the message is valid for.
	//  - roundId - The ID of the round that the message was received on.
	//  - status - the [channels.SentStatus] of the message.
	//
	// Statuses will be enumerated as such:
	//  Sent      =  0
	//  Delivered =  1
	//  Failed    =  2
	//
	// Returns a non-negative unique UUID for the message that it can be
	// referenced by later with [EventModel.UpdateSentStatus].
	ReceiveReply(channelID, messageID, reactionTo []byte,
		nickname, text string, identity []byte,
		timestamp, lease, roundId, status int64) int64

	// ReceiveReaction is called whenever a reaction to a message is received
	// on a given channel. It may be called multiple times on the same reaction.
	// It is incumbent on the user of the API to filter such called by message
	// ID.
	//
	// Messages may arrive our of order, so a reply in theory can arrive before
	// the initial message. As a result, it may be important to buffer
	// reactions.
	//
	// Parameters:
	//  - channelID - The marshalled channel [id.ID].
	//  - messageID - The bytes of the [channel.MessageID] of the received
	//    message.
	//  - reactionTo - The [channel.MessageID] for the message that received a
	//    reply.
	//  - nickname - The nickname of the sender of the message.
	//  - reaction - The contents of the reaction message.
	//  - identity - The json marshal of the identity of the sender
	//  - timestamp - Time the message was received; represented as nanoseconds
	//    since unix epoch.
	//  - lease - The number of nanoseconds that the message is valid for.
	//  - roundId - The ID of the round that the message was received on.
	//  - status - the [channels.SentStatus] of the message.
	//
	// Statuses will be enumerated as such:
	//  Sent      =  0
	//  Delivered =  1
	//  Failed    =  2
	//
	// Returns a non-negative unique uuid for the message by which it can be
	// referenced later with UpdateSentStatus
	ReceiveReaction(channelID, messageID, reactionTo []byte,
		nickname, reaction string, identity []byte,
		timestamp, lease, roundId, status int64) int64

	// UpdateSentStatus is called whenever the sent status of a message has
	// changed.
	//
	// Parameters:
	//  - messageID - The bytes of the [channel.MessageID] of the received
	//    message.
	//  - status - the [channels.SentStatus] of the message.
	//
	// Statuses will be enumerated as such:
	//  Sent      =  0
	//  Delivered =  1
	//  Failed    =  2
	UpdateSentStatus(
		uuid int64, messageID []byte, timestamp, roundID, status int64)

	// unimplemented
	// IgnoreMessage(ChannelID *id.ID, MessageID cryptoChannel.MessageID)
	// UnIgnoreMessage(ChannelID *id.ID, MessageID cryptoChannel.MessageID)
	// PinMessage(ChannelID *id.ID, MessageID cryptoChannel.MessageID, end time.Time)
	// UnPinMessage(ChannelID *id.ID, MessageID cryptoChannel.MessageID)
}

// toEventModel is a wrapper which wraps an existing channels.EventModel object.
type toEventModel struct {
	em EventModel
}

// NewEventModel is a constructor for a toEventModel. This will take in an
// EventModel and wraps it around the toEventModel.
func NewEventModel(em EventModel) channels.EventModel {
	return &toEventModel{em: em}
}

// JoinChannel is called whenever a channel is joined locally.
func (tem *toEventModel) JoinChannel(channel *cryptoBroadcast.Channel) {
	tem.em.JoinChannel(channel.PrettyPrint())
}

// LeaveChannel is called whenever a channel is left locally.
func (tem *toEventModel) LeaveChannel(channelID *id.ID) {
	tem.em.LeaveChannel(channelID[:])
}

// ReceiveMessage is called whenever a message is received on a given channel.
// It may be called multiple times on the same message. It is incumbent on the
// user of the API to filter such called by message ID.
func (tem *toEventModel) ReceiveMessage(channelID *id.ID, messageID cryptoChannel.MessageID,
	nickname, text string, identity cryptoChannel.Identity,
	timestamp time.Time, lease time.Duration, round rounds.Round,
	status channels.SentStatus) uint64 {

	idBytes, err := json.Marshal(&identity)
	if err != nil {
		jww.WARN.Printf("failed to marshal identity object: %+v", err)
		return 0
	}

	return uint64(tem.em.ReceiveMessage(channelID[:], messageID[:], nickname,
		text, idBytes, timestamp.UnixNano(), int64(lease), int64(round.ID),
		int64(status)))
}

// ReceiveReply is called whenever a message is received that is a reply on a
// given channel. It may be called multiple times on the same message. It is
// incumbent on the user of the API to filter such called by message ID.
//
// Messages may arrive our of order, so a reply in theory can arrive before the
// initial message. As a result, it may be important to buffer replies.
func (tem *toEventModel) ReceiveReply(channelID *id.ID, messageID cryptoChannel.MessageID,
	reactionTo cryptoChannel.MessageID, nickname, text string,
	identity cryptoChannel.Identity, timestamp time.Time,
	lease time.Duration, round rounds.Round, status channels.SentStatus) uint64 {

	idBytes, err := json.Marshal(&identity)
	if err != nil {
		jww.WARN.Printf("failed to marshal identity object: %+v", err)
		return 0
	}

	return uint64(tem.em.ReceiveReply(channelID[:], messageID[:], reactionTo[:],
		nickname, text, idBytes, timestamp.UnixNano(), int64(lease),
		int64(round.ID), int64(status)))

}

// ReceiveReaction is called whenever a reaction to a message is received on a
// given channel. It may be called multiple times on the same reaction. It is
// incumbent on the user of the API to filter such called by message ID.
//
// Messages may arrive our of order, so a reply in theory can arrive before the
// initial message. As a result, it may be important to buffer reactions.
func (tem *toEventModel) ReceiveReaction(channelID *id.ID, messageID cryptoChannel.MessageID,
	reactionTo cryptoChannel.MessageID, nickname, reaction string,
	identity cryptoChannel.Identity, timestamp time.Time,
	lease time.Duration, round rounds.Round, status channels.SentStatus) uint64 {

	idBytes, err := json.Marshal(&identity)
	if err != nil {
		jww.WARN.Printf("failed to marshal identity object: %+v", err)
		return 0
	}

	return uint64(tem.em.ReceiveReaction(channelID[:], messageID[:],
		reactionTo[:], nickname, reaction, idBytes, timestamp.UnixNano(),
		int64(lease), int64(round.ID), int64(status)))
}

// UpdateSentStatus is called whenever the sent status of a message has changed.
func (tem *toEventModel) UpdateSentStatus(uuid uint64,
	messageID cryptoChannel.MessageID, timestamp time.Time, round rounds.Round,
	status channels.SentStatus) {
	tem.em.UpdateSentStatus(int64(uuid), messageID[:], timestamp.UnixNano(),
		int64(round.ID), int64(status))
}
