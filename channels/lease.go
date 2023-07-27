////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package channels

import (
	"container/list"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/client/v4/collective/versioned"
	"gitlab.com/elixxir/client/v4/stoppable"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/crypto/message"
	"gitlab.com/xx_network/crypto/randomness"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"io"
	"math/big"
	"strings"
	"time"
)

const (
	// Thread stoppable name
	leaseThreadStoppable = "ActionLeaseThread"

	// Channel sizes
	addLeaseMessageChanSize    = 100
	removeLeaseMessageChanSize = 100
	removeChannelChChanSize    = 100

	// gracePeriod is the minimum amount of time to wait to receive an alternate
	// replay before sending.
	gracePeriod = 3 * time.Minute

	// Range of time to wait for replays to load when loading expired leases.
	replayWaitMin = 5 * time.Minute
	replayWaitMax = 30 * time.Minute

	// Ceiling a floor for quick replay when calling
	// ActionLeaseList.AddOrOverwrite.
	quickReplayFloor   = gracePeriod
	quickReplayCeiling = 10 * time.Minute

	// MessageLife is how long a message is available from the network before it
	// expires from the network and is irretrievable from the gateways.
	MessageLife = 500 * time.Hour

	// leaseNickname is the nickname used when replaying an action.
	leaseNickname = "LeaseSystem"
)

// Error messages.
const (
	// ActionLeaseList.StartProcesses
	noReplayFuncErr = "replay function not registered"
)

// ActionLeaseList keeps a list of messages and actions and undoes each action
// when its lease is up.
type ActionLeaseList struct {
	// List of messages with leases sorted by when their lease ends, smallest to
	// largest.
	leases *list.List

	// List of messages with leases grouped by the channel and keyed on a unique
	// fingerprint.
	messagesByChannel map[id.ID]map[commandFingerprintKey]*leaseMessage

	// New lease messages are added to this channel.
	addLeaseMessage chan *leaseMessagePacket

	// Lease messages that need to be removed are added to this channel.
	removeLeaseMessage chan *leaseMessage

	// Channels that need to be removed are added to this channel.
	removeChannelCh chan *id.ID

	// triggerFn is called when a lease expired to trigger the undoing of the
	// action.
	triggerFn triggerActionEventFunc

	// replayFn is called when a lease is triggered but not expired. It replays
	// the action to extend its life.
	replayFn ReplayActionFunc

	// The replay blocker blocks any replays that occurred in an older round to
	// the currently stored command.
	rb *replayBlocker

	store *CommandStore
	kv    versioned.KV
	rng   *fastRNG.StreamGenerator
}

// ReplayActionFunc replays the encrypted payload on the channel.
type ReplayActionFunc func(channelID *id.ID, encryptedPayload []byte)

// leaseMessagePacket stores the leaseMessage and CommandMessage for moving all
// message data around in memory as a single packet.
type leaseMessagePacket struct {
	*leaseMessage
	cm *CommandMessage
}

// leaseMessage contains a message and an associated action.
type leaseMessage struct {
	// ChannelID is the ID of the channel that his message is in.
	ChannelID *id.ID `json:"channelID"`

	// Action is the action applied to the message (currently only Pinned and
	// Mute).
	Action MessageType `json:"action"`

	// Payload is the contents of the ChannelMessage.Payload.
	Payload []byte `json:"payload"`

	// OriginatingTimestamp is the time the original message was sent. On normal
	// actions, this is the same as Timestamp. On a replayed messages, this is
	// the timestamp of the original message, not the timestamp of the replayed
	// message.
	OriginatingTimestamp time.Time `json:"originatingTimestamp"`

	// Lease is the duration of the message lease. This is the original lease
	// set and indicates the duration to wait from the OriginatingTimestamp.
	Lease time.Duration `json:"lease"`

	// LeaseEnd is the time when the message lease ends. It is the calculated by
	// adding the lease duration to the message's timestamp. Note that LeaseEnd
	// is not when the lease will be triggered.
	LeaseEnd time.Time `json:"leaseEnd"`

	// LeaseTrigger is the time (Unix nano) when the lease is triggered. This is
	// equal to LeaseEnd if Lease is less than MessageLife. Otherwise, this is
	// randomly set between MessageLife/2 and MessageLife.
	LeaseTrigger time.Time `json:"leaseTrigger"`

	// e is a link to this message in the lease list.
	e *list.Element
}

// NewOrLoadActionLeaseList loads an existing ActionLeaseList from storage, if
// it exists. Otherwise, it initialises a new empty ActionLeaseList.
func NewOrLoadActionLeaseList(triggerFn triggerActionEventFunc,
	store *CommandStore, kv versioned.KV, rng *fastRNG.StreamGenerator) (
	*ActionLeaseList, error) {
	all := NewActionLeaseList(triggerFn, store, kv, rng)

	err := all.load(netTime.Now())
	if err != nil && kv.Exists(err) {
		return nil, err
	}

	err = all.rb.load()
	if err != nil && kv.Exists(err) {
		return nil, err
	}

	return all, nil
}

// NewActionLeaseList initialises a new empty ActionLeaseList.
func NewActionLeaseList(triggerFn triggerActionEventFunc, store *CommandStore,
	kv versioned.KV, rng *fastRNG.StreamGenerator) *ActionLeaseList {
	all := &ActionLeaseList{
		leases:             list.New(),
		messagesByChannel:  make(map[id.ID]map[commandFingerprintKey]*leaseMessage),
		addLeaseMessage:    make(chan *leaseMessagePacket, addLeaseMessageChanSize),
		removeLeaseMessage: make(chan *leaseMessage, removeLeaseMessageChanSize),
		removeChannelCh:    make(chan *id.ID, removeChannelChChanSize),
		triggerFn:          triggerFn,
		store:              store,
		kv:                 kv,
		rng:                rng,
	}
	all.rb = newReplayBlocker(all.AddOrOverwrite, store, kv)

	return all
}

// StartProcesses starts the thread that checks for expired action leases and
// undoes the action. This function adheres to the xxdk.Service type.
//
// This function always returns a nil error.
func (all *ActionLeaseList) StartProcesses() (stoppable.Stoppable, error) {
	if all.replayFn == nil {
		return nil, errors.New(noReplayFuncErr)
	}
	actionThreadStop := stoppable.NewSingle(leaseThreadStoppable)

	// Start the thread
	go all.updateLeasesThread(actionThreadStop)

	return actionThreadStop, nil
}

// RegisterReplayFn registers the function that is called to replay an action.
func (all *ActionLeaseList) RegisterReplayFn(replayFn ReplayActionFunc) {
	all.replayFn = replayFn
}

// updateLeasesThread updates the list of message leases and undoes each action
// message when the lease expires.
func (all *ActionLeaseList) updateLeasesThread(stop *stoppable.Single) {
	jww.INFO.Printf(
		"[CH] Starting action lease list thread with stoppable %s", stop.Name())

	// Start timer stopped to wait to receive first message
	var alarmTime time.Duration
	timer := netTime.NewTimer(alarmTime)
	timer.Stop()

	for {
		var lm *leaseMessage

		select {
		case <-stop.Quit():
			jww.INFO.Printf("[CH] Stopping action lease list thread: "+
				"stoppable %s quit", stop.Name())
			stop.ToStopped()
			return
		case lmp := <-all.addLeaseMessage:
			err := all.addMessage(lmp)
			if err != nil {
				jww.FATAL.Panicf("[CH] Failed to add new lease message: %+v", err)
			}
		case lm = <-all.removeLeaseMessage:
			jww.DEBUG.Printf("[CH] Removing lease message: %+v", lm)
			err := all.removeMessage(lm, false)
			if err != nil {
				jww.FATAL.Panicf("[CH] Failed to remove lease message: %+v", err)
			}
		case channelID := <-all.removeChannelCh:
			jww.DEBUG.Printf("[CH] Removing leases for channel %s", channelID)
			err := all.removeChannel(channelID)
			if err != nil {
				jww.FATAL.Panicf("[CH] Failed to remove channel: %+v", err)
			}
		case <-timer.C:
			// Once the timer is triggered, drop below to undo any expired
			// message actions and start the next timer
			jww.DEBUG.Printf("[CH] Lease alarm triggered after %s.", alarmTime)
		}

		timer.Stop()

		// Create list of leases to remove and update so that the list is not
		// modified until after the loop is complete. Otherwise, removing or
		// moving elements during the loop can cause skipping of elements.
		var lmToRemove, lmToUpdate []*leaseMessage

		// Evaluates if the next element in the list needs to be triggered
		activatingNow := func(e *list.Element) bool {
			if e == nil {
				return false
			}

			// Check if the lease trigger is in the past. Subtract a millisecond
			// from the current time to account for clock jitter and/or low
			// resolution clock.
			now := netTime.Now().Add(-time.Millisecond)
			return e.Value.(*leaseMessage).LeaseTrigger.Before(now) ||
				e.Value.(*leaseMessage).LeaseTrigger.Equal(now)
		}

		// loop through all leases which need to be triggered
		e := all.leases.Front()
		for ; activatingNow(e); e = e.Next() {
			lm = e.Value.(*leaseMessage)

			// Load command message from storage
			cm, err :=
				all.store.LoadCommand(lm.ChannelID, lm.Action, lm.Payload)
			if err != nil {
				// If the message cannot be loaded, then skip the trigger or
				// replay and mark for removal
				jww.ERROR.Printf("[CH] Removing lease due to failure to load "+
					"%s command message for channel %s from storage: %+v",
					lm.Action, lm.ChannelID, err)
				lmToRemove = append(lmToRemove, lm)
				continue
			}

			// Check if the real lease has been reached
			if lm.LeaseTrigger.After(lm.LeaseEnd) ||
				lm.LeaseTrigger.Equal(lm.LeaseEnd) {
				// Undo the action of the lease has been reached

				// Mark message for removal
				lmToRemove = append(lmToRemove, lm)
				jww.DEBUG.Printf("[CH] Lease expired at %s; undoing %s for %+v",
					lm.LeaseEnd, lm.Action, lm)

				// Trigger undo
				go func(lm *leaseMessage, cm *CommandMessage) {
					_, err = all.triggerFn(lm.ChannelID, cm.MessageID,
						lm.Action, leaseNickname, lm.Payload,
						cm.EncryptedPayload, cm.Timestamp, lm.OriginatingTimestamp,
						lm.Lease, cm.OriginatingRound, cm.Round, Delivered,
						cm.FromAdmin)
					if err != nil {
						jww.ERROR.Printf("[CH] Failed to trigger %s: %+v",
							lm.Action, err)
					}
				}(lm, cm)
			} else {
				// Replay the message if the real lease has not been reached

				// Mark message for updating
				lmToUpdate = append(lmToUpdate, lm)
				jww.DEBUG.Printf(
					"[CH] Lease triggered at %s; replaying %s for %+v",
					lm.LeaseTrigger, lm.Action, lm)

				// Trigger replay
				go all.replayFn(lm.ChannelID, cm.EncryptedPayload)
			}
		}

		// If there is next lease that has not been reached, set the alarm for
		// next lease trigger
		if e != nil {
			lm = e.Value.(*leaseMessage)
			alarmTime = netTime.Until(lm.LeaseTrigger)
			timer.Reset(alarmTime)

			jww.DEBUG.Printf("[CH] Lease alarm reset for %s for lease %+v",
				alarmTime, lm)
		}

		// Remove all expired actions
		for _, m := range lmToRemove {
			if err := all.removeMessage(m, true); err != nil {
				jww.FATAL.Panicf(
					"[CH] Could not remove lease message: %+v", err)
			}
		}

		// Update all replayed actions
		now := netTime.Now()
		for _, m := range lmToUpdate {
			if err := all.updateLeaseTrigger(m, now); err != nil {
				jww.FATAL.Panicf(
					"[CH] Could not update lease trigger time: %+v", err)
			}
		}

	}
}

// AddMessage triggers the lease message for insertion. An error is returned if
// the message should be dropped. A message is dropped if its lease has expired
// already or if it is older than an already stored replay for the command.
func (all *ActionLeaseList) AddMessage(channelID *id.ID, messageID message.ID,
	action MessageType, unsanitizedPayload, sanitizedPayload,
	encryptedPayload []byte, timestamp, originatingTimestamp time.Time,
	lease time.Duration, originatingRound id.Round, round rounds.Round,
	fromAdmin bool) error {

	// Calculate lease trigger time
	rng := all.rng.GetStream()
	leaseTrigger, leaseActive := calculateLeaseTrigger(
		netTime.Now().UTC().Round(0), originatingTimestamp, lease, rng)
	rng.Close()
	if !leaseActive {
		return errors.Errorf("[CH] Dropping message %s lease for action %s in "+
			"channel %s that has already expired; originatingTimestamp:%s "+
			"lease:%s", messageID, action, channelID, originatingTimestamp,
			lease)
	}

	// Verify that this command, if it is a replay, is newer (i.e. occurred in a
	// newer round) than the currently stored command
	verified, err := all.rb.verifyReplay(channelID, messageID, action,
		unsanitizedPayload, sanitizedPayload, encryptedPayload, timestamp,
		originatingTimestamp, lease, originatingRound, round, fromAdmin)
	if err != nil {
		return errors.Errorf(
			"encountered error when verifying command: %+v", err)
	} else if !verified {
		return errors.New("command replay could not be verified")
	}

	all.addToLeaseMessageChan(channelID, messageID, action, sanitizedPayload,
		encryptedPayload, timestamp, originatingTimestamp, lease,
		originatingRound, round, fromAdmin, leaseTrigger)
	return nil
}

// AddOrOverwrite adds a new lease or overwrites an existing lease to trigger a
// replay soon (between 3 and 10 minutes).
func (all *ActionLeaseList) AddOrOverwrite(channelID *id.ID, action MessageType,
	payload []byte) error {
	// Load command message details from storage
	cm, err := all.store.LoadCommand(channelID, action, payload)
	if err != nil {
		return err
	}

	// Calculate random time between 3 and 10 minutes to send the replay
	rng := all.rng.GetStream()
	leaseDuration := randDurationInRange(
		quickReplayFloor, quickReplayCeiling, rng)
	rng.Close()
	leaseTrigger := netTime.Now().UTC().Round(0).Add(leaseDuration)

	all.addToLeaseMessageChan(channelID, cm.MessageID, action, payload,
		cm.EncryptedPayload, cm.Timestamp, cm.OriginatingTimestamp, cm.Lease,
		cm.OriginatingRound, cm.Round, cm.FromAdmin, leaseTrigger)

	return nil
}

// addMessage inserts the message into the lease list. If the message already
// exists, then its lease is updated.
func (all *ActionLeaseList) addMessage(lmp *leaseMessagePacket) error {

	jww.INFO.Printf("[CH] Inserting new lease: %+v", lmp.leaseMessage)

	// When set to true, the list of channels IDs will be updated in storage
	var channelIdUpdate bool

	fp := newCommandFingerprint(lmp.ChannelID, lmp.Action, lmp.Payload)
	if messages, exists := all.messagesByChannel[*lmp.ChannelID]; !exists {
		// Add the channel if it does not exist
		lmp.e = all.insertLease(lmp.leaseMessage)
		all.messagesByChannel[*lmp.ChannelID] =
			map[commandFingerprintKey]*leaseMessage{fp.key(): lmp.leaseMessage}
		channelIdUpdate = true
	} else if lm, exists2 := messages[fp.key()]; !exists2 {
		// Add the lease message if it does not exist
		lmp.e = all.insertLease(lmp.leaseMessage)
		all.messagesByChannel[*lmp.ChannelID][fp.key()] = lmp.leaseMessage
	} else {
		lm = lmp.leaseMessage
		all.updateLease(lm.e)
	}

	// Update storage
	return all.updateStorage(lmp.ChannelID, channelIdUpdate)
}

// addToLeaseMessageChan constructs the leaseMessagePacket and sends it on the
// new lease message channel.
func (all *ActionLeaseList) addToLeaseMessageChan(channelID *id.ID,
	messageID message.ID, action MessageType, payload, encryptedPayload []byte,
	timestamp, originatingTimestamp time.Time, lease time.Duration,
	originatingRound id.Round, round rounds.Round, fromAdmin bool,
	leaseTrigger time.Time) {
	all.addLeaseMessage <- &leaseMessagePacket{
		leaseMessage: &leaseMessage{
			ChannelID:            channelID,
			Action:               action,
			Payload:              payload,
			OriginatingTimestamp: originatingTimestamp,
			Lease:                lease,
			LeaseEnd:             originatingTimestamp.Add(lease),
			LeaseTrigger:         leaseTrigger,
			e:                    nil,
		},
		cm: &CommandMessage{
			ChannelID:            channelID,
			MessageID:            messageID,
			MessageType:          action,
			Content:              payload,
			EncryptedPayload:     encryptedPayload,
			Timestamp:            timestamp,
			OriginatingTimestamp: originatingTimestamp,
			Lease:                lease,
			OriginatingRound:     originatingRound,
			Round:                round,
			FromAdmin:            fromAdmin,
		},
	}
}

// insertLease inserts the leaseMessage to the lease list in order and returns
// the element in the list. Returns true if it was added to the head of the
// list.
func (all *ActionLeaseList) insertLease(lm *leaseMessage) *list.Element {
	mark := all.findSortedPosition(lm.LeaseTrigger)
	if mark == nil {
		return all.leases.PushBack(lm)
	} else {
		return all.leases.InsertBefore(lm, mark)
	}
}

// updateLease updates the location of the given element. This should be called
// when the LeaseTrigger for a message changes. Returns true if it was added to
// the head of the list.
func (all *ActionLeaseList) updateLease(e *list.Element) {
	mark := all.findSortedPosition(e.Value.(*leaseMessage).LeaseTrigger)
	if mark == nil {
		all.leases.MoveToBack(e)
	} else {
		all.leases.MoveBefore(e, mark)
	}
}

// findSortedPosition finds the location in the list where the lease trigger can
// be inserted and returns the next element.
//
// Note: A find operations has an O(n).
func (all *ActionLeaseList) findSortedPosition(leaseTrigger time.Time) *list.Element {
	for mark := all.leases.Front(); mark != nil; mark = mark.Next() {
		if mark.Value.(*leaseMessage).LeaseTrigger.After(leaseTrigger) {
			return mark
		}
	}
	return nil
}

// RemoveMessage triggers the lease message for removal. An error is returned if
// the message should be dropped. A message is dropped if its lease has expired
// already or if it is older than an already stored replay for the command.
func (all *ActionLeaseList) RemoveMessage(channelID *id.ID,
	messageID message.ID, action MessageType, unsanitizedPayload,
	sanitizedPayload, encryptedPayload []byte, timestamp,
	originatingTimestamp time.Time, lease time.Duration,
	originatingRound id.Round, round rounds.Round, fromAdmin bool) error {

	// Reject commands with expired leases
	if netTime.Now().Sub(originatingTimestamp) >= lease {
		return errors.New("lease already expired")
	}

	// Verify that this command, if it is a replay, is newer (i.e. occurred in a
	// newer round) than the currently stored command
	verified, err := all.rb.verifyReplay(channelID, messageID, action,
		unsanitizedPayload, sanitizedPayload, encryptedPayload, timestamp,
		originatingTimestamp, lease, originatingRound, round, fromAdmin)
	if err != nil {
		return errors.Errorf(
			"encountered error when verifying command: %+v", err)
	} else if !verified {
		return errors.New("command replay could not be verified")
	}

	all.removeLeaseMessage <- &leaseMessage{
		ChannelID: channelID,
		Action:    action,
		Payload:   sanitizedPayload,
	}

	return nil
}

// removeMessage removes the lease message from the lease list and the message
// map. This function also updates storage. If the message does not exist, nil
// is returned. Set leaseExpired to true if the message is being removed because
// its lease expired.
func (all *ActionLeaseList) removeMessage(
	newLm *leaseMessage, leaseExpired bool) error {
	fp := newCommandFingerprint(newLm.ChannelID, newLm.Action, newLm.Payload)
	var lm *leaseMessage
	if messages, exists := all.messagesByChannel[*newLm.ChannelID]; !exists {
		return nil
	} else if lm, exists = messages[fp.key()]; !exists {
		return nil
	}

	// Remove from lease list
	all.leases.Remove(lm.e)

	// When set to true, the list of channels IDs will be updated in storage
	var channelIdUpdate bool

	// Remove from message map
	delete(all.messagesByChannel[*lm.ChannelID], fp.key())
	if len(all.messagesByChannel[*lm.ChannelID]) == 0 {
		delete(all.messagesByChannel, *lm.ChannelID)
		channelIdUpdate = true
	}

	// Remove from replay blocker if the lease has expired
	if leaseExpired {
		// FIXME: If a command is undone before its lease expires, it will
		//  remain in the replay blocker forever. The lease system should be
		//  modified to remove the command from the replay blocker when the
		//  lease expired even if it has been undone.
		err := all.rb.removeCommand(lm.ChannelID, lm.Action, lm.Payload)
		if err != nil {
			jww.ERROR.Printf("[CH] Failed to delete command %s for channel %s "+
				"from replay blocker: %+v", lm.Action, lm.ChannelID, err)
		}

		// Delete from command storage
		err = all.store.DeleteCommand(lm.ChannelID, lm.Action, lm.Payload)
		if err != nil {
			jww.ERROR.Printf("[CH] Failed to delete command %s for channel %s "+
				"from storage: %+v", lm.Action, lm.ChannelID, err)
		}
	}

	// Update storage
	return all.updateStorage(lm.ChannelID, channelIdUpdate)
}

// updateLeaseTrigger updates the lease trigger time for the given lease
// message. This function also updates storage. If the message does not exist,
// nil is returned.
func (all *ActionLeaseList) updateLeaseTrigger(
	newLm *leaseMessage, now time.Time) error {
	fp := newCommandFingerprint(newLm.ChannelID, newLm.Action, newLm.Payload)
	var lm *leaseMessage
	if messages, exists := all.messagesByChannel[*newLm.ChannelID]; !exists {
		jww.WARN.Printf("[CH] Could not find channel %s in lease system for "+
			"key %s to update trigger. This should not happen and indicates "+
			"a bug in the channels lease code.", newLm.ChannelID, fp)
		return nil
	} else if lm, exists = messages[fp.key()]; !exists {
		jww.WARN.Printf("[CH] Could not find lease message in channel %s and "+
			"key %s to update trigger. This should not happen and indicates "+
			"a bug in the channels lease code.", newLm.ChannelID, fp)
		return nil
	}

	// Calculate random trigger duration
	rng := all.rng.GetStream()
	leaseTrigger, leaseActive :=
		calculateLeaseTrigger(now, lm.OriginatingTimestamp, lm.Lease, rng)
	rng.Close()
	if !leaseActive {
		return all.removeMessage(lm, true)
	}

	all.messagesByChannel[*newLm.ChannelID][fp.key()].LeaseTrigger = leaseTrigger
	all.updateLease(lm.e)

	// Update storage
	return all.updateStorage(lm.ChannelID, false)
}

// RemoveChannel triggers all leases for the channel for removal.
func (all *ActionLeaseList) RemoveChannel(channelID *id.ID) {
	all.removeChannelCh <- channelID
}

// removeChannel removes each lease message for the channel from the leases
// list and removes the channel from the messages map. Also deletes from
// storage.
func (all *ActionLeaseList) removeChannel(channelID *id.ID) error {
	leases, exists := all.messagesByChannel[*channelID]
	if !exists {
		return nil
	}

	for _, lm := range leases {
		all.leases.Remove(lm.e)
		err := all.store.DeleteCommand(lm.ChannelID, lm.Action, lm.Payload)
		if err != nil {
			jww.ERROR.Printf("[CH] Failed to delete command %s for channel %s "+
				"from storage: %+v", lm.Action, lm.ChannelID, err)
		}
	}

	err := all.rb.removeChannelCommands(channelID)
	if err != nil {
		jww.ERROR.Printf("[CH] Failed to delete commands from replay blocker "+
			"for channel %s: %+v", channelID, err)
	}

	delete(all.messagesByChannel, *channelID)

	err = all.storeLeaseChannels()
	if err != nil {
		return err
	}

	return all.deleteLeaseMessages(channelID)
}

// calculateLeaseTrigger calculates the time until the lease should be
// triggered. If the lease is smaller than MessageLife, then its lease will be
// triggered when the lease is reached. If the lease is greater than
// MessageLife, then the message will need to be replayed and its lease will be
// triggered at some random time between half of MessageLife and MessageLife.
//
// If the lease has already been reached or will be before the gracePeriod is
// reached, the message should be dropped and calculateLeaseTrigger returns
// false. Otherwise, it returns the lease trigger and true.
func calculateLeaseTrigger(now, originatingTimestamp time.Time,
	lease time.Duration, rng io.Reader) (time.Time, bool) {
	elapsedLife := now.Sub(originatingTimestamp)

	if elapsedLife >= lease {
		// If the lease has already been reached, drop the message
		return time.Time{}, false
	} else if lease == ValidForever || lease-elapsedLife >= MessageLife {
		// If the message lasts forever or the lease extends longer than a
		// message life, then it needs to be replayed
		lease = MessageLife
		originatingTimestamp = now
	} else {
		// If the lease is smaller than MessageLife, than the lease trigger is
		// the same as the lease end
		return originatingTimestamp.Add(lease), true
	}

	// Calculate the floor to be half of the lease life. If that is in the past,
	// then the floor is set to the current time (plus a grace period to ensure
	// no other leases are received).
	floor := originatingTimestamp.Add(lease / 2)
	if now.After(floor) {
		floor = now.Add(gracePeriod)
	}

	// Set the ceiling to the end of the lease
	ceiling := originatingTimestamp.Add(lease)

	// Drop the message if the ceiling occurs before the grace period or the
	// message is about to expire
	if floor.After(ceiling) || ceiling.Sub(floor) < gracePeriod {
		return time.Time{}, false
	}

	// Generate random duration between floor and ceiling
	lease = randDurationInRange(0, ceiling.Sub(floor), rng)

	return floor.Add(lease), true
}

// randDurationInRange generates a random positive int64 between start and end.
func randDurationInRange(start, end time.Duration, rng io.Reader) time.Duration {
	// Generate 256-bit seed
	const seedSize = 32
	seed := make([]byte, seedSize)
	if n, err := rng.Read(seed); err != nil {
		jww.FATAL.Panicf("[CH] Failed to generate random seed: %+v", err)
	} else if n != 32 {
		jww.FATAL.Panicf("[CH] Generated %d bytes for random seed when %d "+
			"bytes are required.", n, seedSize)
	}

	h, err := hash.NewCMixHash()
	if err != nil {
		jww.FATAL.Panicf("[CH] Failed to initialize new hash: %+v", err)
	}

	n := randomness.RandInInterval(big.NewInt(int64(end-start)), seed, h)

	return start + time.Duration(n.Int64())
}

// String prints the leaseMessagePacket in a human-readable form for logging and
// debugging. This function adheres to the fmt.Stringer interface.
func (lmp *leaseMessagePacket) String() string {
	fields := []string{
		"leaseMessage:" + lmp.leaseMessage.String(),
		"CommandMessage:" + fmt.Sprintf("%+v", lmp.cm),
	}

	return "{" + strings.Join(fields, " ") + "}"
}

// String prints the leaseMessage in a human-readable form for logging and
// debugging. This function adheres to the fmt.Stringer interface.
func (lm *leaseMessage) String() string {
	trunc := func(b []byte, n int) string {
		if len(b) <= n-3 {
			return hex.EncodeToString(b)
		} else {
			return hex.EncodeToString(b[:n]) + "..."
		}
	}

	fields := []string{
		"ChannelID:" + lm.ChannelID.String(),
		"Action:" + lm.Action.String(),
		"Payload:" + trunc(lm.Payload, 6),
		"OriginatingTimestamp:" + lm.OriginatingTimestamp.String(),
		"Lease:" + lm.Lease.String(),
		"LeaseEnd:" + lm.LeaseEnd.String(),
		"LeaseTrigger:" + lm.LeaseTrigger.String(),
		"e:" + fmt.Sprintf("%p", lm.e),
	}

	return "{" + strings.Join(fields, " ") + "}"
}

////////////////////////////////////////////////////////////////////////////////
// Storage Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

// Storage values.
const (
	channelLeaseVer               = 0
	channelLeaseKey               = "channelLeases"
	channelLeaseMessagesVer       = 0
	channelLeaseMessagesKeyPrefix = "channelLeaseMessages/"
)

// Error messages.
const (
	// ActionLeaseList.load
	loadLeaseChanIDsErr  = "could not load list of channels"
	loadLeaseMessagesErr = "could not load message leases for channel %s"

	// ActionLeaseList.updateStorage
	storeLeaseMessagesErr = "could not store message leases for channel %s: %+v"
	storeLeaseChanIDsErr  = "could not store lease channel IDs: %+v"
)

// load gets all the lease messages from storage and loads them into the lease
// list and message map. If any of the lease messages have a lease trigger
// before now, then they are assigned a new lease trigger between 5 and 30
// minutes in the future to allow alternate replays a chance to be picked up.
func (all *ActionLeaseList) load(now time.Time) error {
	// Get list of channel IDs
	channelIDs, err := all.loadLeaseChannels()
	if err != nil {
		return errors.Wrap(err, loadLeaseChanIDsErr)
	}

	// Get list of lease messages and load them into the message map and lease
	// list
	rng := all.rng.GetStream()
	defer rng.Close()
	// FIXME: This has a hidden sort of O(n^2) because the insert done in each
	//  iteration is O(n). This should be improved. Quicksort first and then
	//  insert (do not use insertLease).
	for _, channelID := range channelIDs {
		all.messagesByChannel[*channelID], err = all.loadLeaseMessages(channelID)
		if err != nil {
			return errors.Wrapf(err, loadLeaseMessagesErr, channelID)
		}

		for _, lm := range all.messagesByChannel[*channelID] {
			// Check if the lease has expired
			if lm.LeaseTrigger.Before(now) {
				waitForReplayDuration :=
					randDurationInRange(replayWaitMin, replayWaitMax, rng)
				if lm.LeaseTrigger == lm.LeaseEnd {
					lm.LeaseEnd = now.Add(waitForReplayDuration)
				}
				lm.LeaseTrigger = now.Add(waitForReplayDuration)
			}

			lm.e = all.insertLease(lm)
		}
	}

	return nil
}

// updateStorage updates the given channel lease list in storage. If
// channelIdUpdate is true, then the main list of channel IDs is also updated.
// Use this option when adding or removing a channel ID from the message map.
func (all *ActionLeaseList) updateStorage(
	channelID *id.ID, channelIdUpdate bool) error {
	if err := all.storeLeaseMessages(channelID); err != nil {
		return errors.Errorf(storeLeaseMessagesErr, channelID, err)
	} else if channelIdUpdate {
		if err = all.storeLeaseChannels(); err != nil {
			return errors.Errorf(storeLeaseChanIDsErr, err)
		}
	}
	return nil
}

// storeLeaseChannels stores the list of all channel IDs in the lease list to
// storage.
func (all *ActionLeaseList) storeLeaseChannels() error {
	channelIDs := make([]*id.ID, 0, len(all.messagesByChannel))
	for chanID := range all.messagesByChannel {
		channelIDs = append(channelIDs, chanID.DeepCopy())
	}

	data, err := json.Marshal(&channelIDs)
	if err != nil {
		return err
	}

	obj := &versioned.Object{
		Version:   channelLeaseVer,
		Timestamp: netTime.Now(),
		Data:      data,
	}

	return all.kv.Set(channelLeaseKey, obj)
}

// loadLeaseChannels loads the list of all channel IDs in the lease list from
// storage.
func (all *ActionLeaseList) loadLeaseChannels() ([]*id.ID, error) {
	obj, err := all.kv.Get(channelLeaseKey, channelLeaseVer)
	if err != nil {
		return nil, err
	}

	var channelIDs []*id.ID
	return channelIDs, json.Unmarshal(obj.Data, &channelIDs)
}

// storeLeaseMessages stores the list of leaseMessage objects for the given
// channel ID to storage keying on the channel ID.
func (all *ActionLeaseList) storeLeaseMessages(channelID *id.ID) error {
	// If the list is empty, then delete it from storage
	if len(all.messagesByChannel[*channelID]) == 0 {
		return all.deleteLeaseMessages(channelID)
	}

	data, err := json.Marshal(all.messagesByChannel[*channelID])
	if err != nil {
		return err
	}

	obj := &versioned.Object{
		Version:   channelLeaseMessagesVer,
		Timestamp: netTime.Now(),
		Data:      data,
	}

	return all.kv.Set(makeChannelLeaseMessagesKey(channelID), obj)
}

// loadLeaseMessages loads the list of leaseMessage from storage keyed on the
// channel ID.
func (all *ActionLeaseList) loadLeaseMessages(channelID *id.ID) (
	map[commandFingerprintKey]*leaseMessage, error) {
	obj, err := all.kv.Get(
		makeChannelLeaseMessagesKey(channelID), channelLeaseMessagesVer)
	if err != nil {
		return nil, err
	}

	var messages map[commandFingerprintKey]*leaseMessage
	return messages, json.Unmarshal(obj.Data, &messages)
}

// deleteLeaseMessages deletes the list of leaseMessage from storage that is
// keyed on the channel ID.
func (all *ActionLeaseList) deleteLeaseMessages(channelID *id.ID) error {
	return all.kv.Delete(
		makeChannelLeaseMessagesKey(channelID), channelLeaseMessagesVer)
}

// makeChannelLeaseMessagesKey creates a key for saving channel lease messages
// to storage.
func makeChannelLeaseMessagesKey(channelID *id.ID) string {
	return channelLeaseMessagesKeyPrefix +
		base64.StdEncoding.EncodeToString(channelID.Marshal())
}
