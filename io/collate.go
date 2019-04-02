////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"crypto/sha256"
	"fmt"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/client/user"
	"gitlab.com/elixxir/primitives/format"
	"sync"
	"time"
)

type multiPartMessage struct {
	parts            [][]byte
	numPartsReceived uint8
}

const PendingMessageKeyLenBits = uint64(256)
const PendingMessageKeyLen = PendingMessageKeyLenBits / 8

type PendingMessageKey [PendingMessageKeyLen]byte

type collator struct {
	pendingMessages map[PendingMessageKey]*multiPartMessage
	// TODO do we need a lock here? or can we assume that requests will come
	// from only one thread?
	mux sync.Mutex
}

var theCollator *collator

func GetCollator() *collator {
	if theCollator == nil {
		theCollator = &collator{
			pendingMessages: make(map[PendingMessageKey]*multiPartMessage),
		}
	}
	return theCollator
}

// AddMessage validates its input and silently does nothing on failure
// TODO should this return an error?
// TODO this should take a different type as parameter.
// TODO this takes too many types. i should split it up.
// This method returns a byte slice with the assembled message if it's
// received a completed message.
func (mb *collator) AddMessage(message *format.Message,
	timeout time.Duration) *parse.Message {

	payload := message.GetPayloadData()
	sender := message.GetSender()
	recipient := message.GetRecipient()

	partition, err := parse.ValidatePartition(payload)

	if err == nil {
		if partition.MaxIndex == 0 {
			//this is the only part of the message. we should take the fast
			//path and skip putting it in the map
			typedBody, err := parse.Parse(partition.Body)
			// Log an error if the message is malformed and return nothing
			if err != nil {
				globals.Log.ERROR.Printf("Malformed message recieved")
				return nil
			}

			msg := parse.Message{
				TypedBody: *typedBody,
				OuterType: format.Unencrypted,
				Sender:    sender,
				Receiver:  user.TheSession.GetCurrentUser().User,
			}

			return &msg
		} else {
			// assemble the map key into a new chunk of memory
			var key PendingMessageKey
			h := sha256.New()
			h.Write(partition.ID)
			h.Write(sender.Bytes())
			keyHash := h.Sum(nil)
			copy(key[:], keyHash[:PendingMessageKeyLen])

			mb.mux.Lock()
			message, ok := mb.pendingMessages[key]
			if !ok {
				// this is a multi-part message we haven't seen before.
				// make a new array of partitions for this key
				newMessage := make([][]byte, partition.MaxIndex+1)
				newMessage[partition.Index] = partition.Body

				message = &multiPartMessage{
					parts:            newMessage,
					numPartsReceived: 1,
				}

				mb.pendingMessages[key] = message

				// start timeout for these partitions
				// TODO vary timeout depending on number of messages?
				time.AfterFunc(timeout, func() {
					mb.mux.Lock()
					_, ok := mb.pendingMessages[key]
					if ok {
						delete(mb.pendingMessages, key)
					}
					mb.mux.Unlock()
				})
			} else {
				// append to array for this key
				message.numPartsReceived++
				message.parts[partition.Index] = partition.Body
			}
			if message.numPartsReceived > partition.MaxIndex {
				// Construct message
				fullMsg, err := parse.Assemble(message.parts)
				if err != nil {
					delete(mb.pendingMessages, key)
					mb.mux.Unlock()
					globals.Log.ERROR.Printf("Malformed message: Padding error, %v", err.Error())
					return nil
				}
				typedBody, err := parse.Parse(fullMsg)
				// Log an error if the message is malformed and return nothing
				if err != nil {
					delete(mb.pendingMessages, key)
					mb.mux.Unlock()
					globals.Log.ERROR.Printf("Malformed message Received")
					return nil
				}

				msg := parse.Message{
					TypedBody: *typedBody,
					OuterType: format.Unencrypted,
					Sender:    sender,
					Receiver:  recipient,
				}

				delete(mb.pendingMessages, key)
				mb.mux.Unlock()
				return &msg
			}
			mb.mux.Unlock()
		}
	} else {
		globals.Log.ERROR.Printf("Received an invalid partition: %v\n", err.Error())
	}
	globals.Log.DEBUG.Printf("Message collator: %v", mb.dump())
	return nil
}

// Debug: dump all messages that are currently in the map
func (mb *collator) dump() string {
	dump := ""
	mb.mux.Lock()
	for key := range mb.pendingMessages {
		if mb.pendingMessages[key].parts != nil {
			for i, part := range mb.pendingMessages[key].parts {
				dump += fmt.Sprintf("Part %v: %v\n", i, part)
			}
			dump += fmt.Sprintf("Total parts received: %v\n",
				mb.pendingMessages[key].numPartsReceived)
		}
	}
	mb.mux.Unlock()
	return dump
}
