package io

import (
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/privategrity/client/parse"
	"gitlab.com/privategrity/client/user"
	"sync"
	"time"
)

type multiPartMessage struct {
	parts            [][]byte
	numPartsReceived uint8
}

type collator struct {
	pendingMessages map[string]*multiPartMessage
	// TODO do we need a lock here? or can we assume that requests will come
	// from only one thread?
	mux sync.Mutex
}

var theCollator *collator

func GetCollator() *collator {
	if theCollator == nil {
		theCollator = &collator{
			pendingMessages: make(map[string]*multiPartMessage),
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
func (mb *collator) AddMessage(payload []byte, sender user.ID,
	timeout time.Duration) []byte {
	//fmt.Printf("Adding a message from %v to the collator", sender)
	partition, err := parse.ValidatePartition(payload)

	if err == nil {
		if partition.MaxIndex == 0 {
			//this is the only part of the message. we should take the fast
			//path and skip putting it in the map
			return partition.Body
		} else {
			// TODO hash something here for better security properties?
			// assemble the map key into a new chunk of memory
			senderBytes := sender.Bytes()
			keyBytes := make([]byte, len(partition.ID)+len(senderBytes))
			copiedBytes := copy(keyBytes, partition.ID)
			copy(keyBytes[copiedBytes:], senderBytes)
			key := string(keyBytes)

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
				// TODO broadcastMessageReception should maybe take a container
				// type. something like format. MessageInterface? or parse.Message?
				assembledMessages := parse.Assemble(message.parts)
				delete(mb.pendingMessages, key)
				mb.mux.Unlock()
				return assembledMessages
			}
			mb.mux.Unlock()
		}
	} else {
		jww.ERROR.Printf("Received an invalid partition: %v\n", err.Error())
	}
	jww.DEBUG.Printf("Message collator: %v", mb.dump())
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
