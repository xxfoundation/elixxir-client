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
	cryptoBroadcast "gitlab.com/elixxir/crypto/broadcast"
	cryptoChannel "gitlab.com/elixxir/crypto/channel"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"sync"
	"time"
)

////////////////////////////////////////////////////////////////////////////////
// Singleton Tracker                                                          //
////////////////////////////////////////////////////////////////////////////////

// eventModelTrackerSingleton is used to track EventModel objects
// so that they can be referenced by ID back over the bindings.
var eventModelTrackerSingleton = &eventModelTracker{
	tracked: make(map[int]*EventModel),
	count:   0,
}

// eventModelTracker is a singleton used to keep track of extant
// EventModel objects, preventing race conditions created by passing it
// over the bindings.
type eventModelTracker struct {
	tracked map[int]*EventModel
	count   int
	mux     sync.RWMutex
}

// make create an EventModel from an [channels.EventModel], assigns it a
// unique ID, and adds it to the eventModelTracker.
func (emt *eventModelTracker) make(eventModel channels.EventModel) *EventModel {
	emt.mux.Lock()
	defer emt.mux.Unlock()

	id := emt.count
	emt.count++

	emt.tracked[id] = &EventModel{
		api: eventModel,
		id:  id,
	}

	return emt.tracked[id]
}

// get an EventModel from the eventModelTracker given its ID.
func (emt *eventModelTracker) get(id int) (*EventModel, error) {
	emt.mux.RLock()
	defer emt.mux.RUnlock()

	c, exist := emt.tracked[id]
	if !exist {
		return nil, errors.Errorf(
			"Cannot get EventModel for ID %d, does not exist", id)
	}

	return c, nil
}

// delete removes a EventModel from the eventModelTracker.
func (emt *eventModelTracker) delete(id int) {
	emt.mux.Lock()
	defer emt.mux.Unlock()

	delete(emt.tracked, id)
}

////////////////////////////////////////////////////////////////////////////////
// Basic EventModel API                                                          //
////////////////////////////////////////////////////////////////////////////////

type EventModel struct {
	api channels.EventModel
	id  int
}

// NewEventModel IS CURRENTLY UNIMPLEMENTED.
func NewEventModel() *EventModel {
	return eventModelTrackerSingleton.make(nil)
}

// JoinChannel is called whenever a channel is joined locally.
//
// Parameters:
//  - channelJson - A JSON encoded [ChannelDef].
func (e *EventModel) JoinChannel(channelJson []byte) {
	// Unmarshal channel definition
	def := ChannelDef{}
	err := json.Unmarshal(channelJson, &def)
	if err != nil {
		jww.ERROR.Printf("Could not parse channel JSON: %+v", err)
		return
	}

	// Construct ID using the embedded cryptographic information
	channelId, err := cryptoBroadcast.NewChannelID(def.Name, def.Description,
		def.Salt, def.PubKey)
	if err != nil {
		jww.ERROR.Printf("Could not construct channel ID: %+v", err)
		return
	}

	// Construct public key into object
	rsaPubKey, err := rsa.LoadPublicKeyFromPem(def.PubKey)
	if err != nil {
		jww.ERROR.Printf("Could not read public key: %+v", err)
		return
	}

	// Construct cryptographic channel object
	channel := &cryptoBroadcast.Channel{
		ReceptionID: channelId,
		Name:        def.Name,
		Description: def.Description,
		Salt:        def.Salt,
		RsaPubKey:   rsaPubKey,
	}

	e.api.JoinChannel(channel)
	return
}

// LeaveChannel is called whenever a channel is left locally.
//
// Parameters:
//  - []byte - A JSON marshalled channel ID ([id.ID]). This may be retrieved
//    using ChannelsManager.GetChannelId.
func (e *EventModel) LeaveChannel(marshalledChanId []byte) {
	// Unmarshal channel ID
	channelId, err := id.Unmarshal(marshalledChanId)
	if err != nil {
		jww.ERROR.Printf("Could not parse channel ID (%s): %+v",
			marshalledChanId, err)
		return
	}

	e.api.LeaveChannel(channelId)
	return
}

// ReceiveMessage is called whenever a message is received on a given channel
// It may be called multiple times on the same message, it is incumbent on
// the user of the API to filter such called by message ID.
//
// Parameters:
//  - reportJson - A JSON marshalled ReceivedChannelMessageReport.
func (e *EventModel) ReceiveMessage(reportJson []byte) {
	report, err := parseChannelMessageReport(reportJson)
	if err != nil {
		jww.ERROR.Printf("%+v", err)
		return
	}

	// fixme: the internal API should accept an object, probably
	//  just use receivedChannelMessageReport in the channels package
	e.api.ReceiveMessage(report.ChannelID, report.MessageID,
		report.SenderUsername, report.Content, report.Timestamp,
		report.Lease, report.Round)

	return
}

// ReceiveReply is called whenever a message is received which is a reply
// on a given channel. It may be called multiple times on the same message,
// it is incumbent on the user of the API to filter such called by message ID
// Messages may arrive our of order, so a reply in theory can arrive before
// the initial message, as a result it may be important to buffer replies.
//
// Parameters:
//  - reportJson - A JSON marshalled ReceivedChannelMessageReport.
func (e *EventModel) ReceiveReply(reportJson []byte) {
	report, err := parseChannelMessageReport(reportJson)
	if err != nil {
		jww.ERROR.Printf("%+v", err)
		return
	}

	// fixme: the internal API should accept an object, probably
	//  just use receivedChannelMessageReport in the channels package. This i
	e.api.ReceiveReply(report.ChannelID, report.MessageID, report.ReplyTo,
		report.SenderUsername, report.Content, report.Timestamp,
		report.Lease, report.Round)
}

// ReceiveReaction is called whenever a Content to a message is received
// on a given channel. It may be called multiple times on the same Content,
// it is incumbent on the user of the API to filter such called by message ID
// Messages may arrive our of order, so a reply in theory can arrive before
// the initial message, as a result it may be important to buffer reactions.
//
// Parameters:
//  - reportJson - A JSON marshalled ReceivedChannelMessageReport.
func (e *EventModel) ReceiveReaction(reportJson []byte) {
	report, err := parseChannelMessageReport(reportJson)
	if err != nil {
		jww.ERROR.Printf("%+v", err)
		return
	}

	// fixme: the internal API should accept an object, probably
	//  just move receivedChannelMessageReport to the channels package and export it.
	e.api.ReceiveReaction(report.ChannelID, report.MessageID, report.ReplyTo,
		report.SenderUsername, report.Content, report.Timestamp, report.Lease,
		report.Round)
}

// receivedChannelMessageReport is the Golang representation of
// a channel message report.
type receivedChannelMessageReport struct {

	// Channel ID is the message of the channel this message was received on.
	ChannelID *id.ID

	// MessageID is the ID of the channel message received.
	MessageID cryptoChannel.MessageID

	// ReplyTo is overloaded to be a reply or react to,
	// depending on the context of the received message
	// (EventModel.ReceiveReaction or EventModel.ReceiveReply).
	ReplyTo cryptoChannel.MessageID

	// SenderUsername is the username of the sender.
	SenderUsername string

	// Content is the payload of the message. This is overloaded with
	// reaction in the [EventModel.ReceiveReaction]. This may
	// either be text or an emoji.
	Content string

	// The timestamp of the message.
	Timestamp time.Time

	// The duration of this channel message.
	Lease time.Duration

	// The round this message was sent on.
	Round rounds.Round
}

// parseChannelMessageReport converts the JSON representation of
// a ReceivedChannelMessageReport into the Golang representation,
// receivedChannelMessageReport.
func parseChannelMessageReport(reportJson []byte) (
	receivedChannelMessageReport, error) {

	// Unmarshal message report
	messageReport := ReceivedChannelMessageReport{}
	err := json.Unmarshal(reportJson, &messageReport)
	if err != nil {
		return receivedChannelMessageReport{},
			errors.Errorf("Could not parse received message report (%s): %+v",
				reportJson, err)
	}

	// Unmarshal channel ID
	chanId, err := id.Unmarshal(messageReport.ChannelId)
	if err != nil {
		return receivedChannelMessageReport{},
			errors.Errorf("Could not parse channel ID (%s): %+v",
				messageReport.ChannelId, err)
	}

	// Unmarshal message ID
	msgId := cryptoChannel.MessageID{}
	copy(msgId[:], messageReport.MessageId)

	// Unmarshal reply to/react to message ID
	replyTo := cryptoChannel.MessageID{}
	copy(replyTo[:], messageReport.ReplyTo)

	// Construct Round
	rnd := rounds.Round{ID: id.Round(messageReport.Rounds[0])}

	return receivedChannelMessageReport{
		ChannelID:      chanId,
		MessageID:      msgId,
		ReplyTo:        replyTo,
		SenderUsername: messageReport.SenderUsername,
		Content:        string(messageReport.Content),
		Timestamp:      time.Unix(0, messageReport.Timestamp),
		Lease:          time.Duration(messageReport.Lease),
		Round:          rnd,
	}, nil

}
