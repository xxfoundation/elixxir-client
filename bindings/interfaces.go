////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/primitives/switchboard"
)

// Message used for binding
type Message interface {
	// Returns the message's sender ID
	GetSender() []byte
	// Returns the message payload
	// Parse this with protobuf/whatever according to the type of the message
	GetPayload() []byte
	// Returns the message's recipient ID
	GetRecipient() []byte
	// Returns the message's type
	GetMessageType() int32
	// Returns the message's timestamp in seconds since unix epoc
	GetTimestamp() int64
	// Returns the message's timestamp in ns since unix epoc
	GetTimestampNano() int64
}

// Copy of the storage interface.
// It is identical to the interface used in Globals,
// and a results the types can be passed freely between the two
type Storage interface {
	// Give a Location for storage.  Does not need to be implemented if unused.
	SetLocation(string, string) error
	// Returns the Location for storage.
	// Does not need to be implemented if unused.
	GetLocation() string
	// Stores the passed byte slice to location A
	SaveA([]byte) error
	// Returns the stored byte slice stored in location A
	LoadA() []byte
	// Stores the passed byte slice to location B
	SaveB([]byte) error
	// Returns the stored byte slice stored in location B
	LoadB() []byte
	// Returns whether the storage has even been written to.
	// if something exists in A or B
	IsEmpty() bool
}

// Translate a bindings storage to a client storage
type storageProxy struct {
	boundStorage Storage
}

// Translate a bindings message to a parse message
// An object implementing this interface can be called back when the client
// gets a message of the type that the registerer specified at registration
// time.
type Listener interface {
	Hear(msg Message, isHeardElsewhere bool, i ...interface{})
}

// Translate a bindings listener to a switchboard listener
// Note to users of this package from other languages: Symbols that start with
// lowercase are unexported from the package and meant for internal use only.
type listenerProxy struct {
	proxy Listener
}

func (lp *listenerProxy) Hear(msg switchboard.Item, isHeardElsewhere bool, i ...interface{}) {
	msgInterface := &parse.BindingsMessageProxy{Proxy: msg.(*parse.Message)}
	lp.proxy.Hear(msgInterface, isHeardElsewhere, i)
}

// Interface used to receive a callback on searching for a user
type SearchCallback interface {
	Callback(userID, pubKey []byte, err error)
}

type searchCallbackProxy struct {
	proxy SearchCallback
}

func (scp *searchCallbackProxy) Callback(userID, pubKey []byte, err error) {
	scp.proxy.Callback(userID, pubKey, err)
}

// Interface used to receive a callback on searching for a user's nickname
type NickLookupCallback interface {
	Callback(nick string, err error)
}

type nickCallbackProxy struct {
	proxy NickLookupCallback
}

// interface used to receive the result of a nickname request
func (ncp *nickCallbackProxy) Callback(nick string, err error) {
	ncp.proxy.Callback(nick, err)
}

// interface used to receive a ui friendly description of the current status of
// registration
type ConnectionStatusCallback interface {
	Callback(status int, TimeoutSeconds int)
}

type OperationProgressCallback interface {
	Callback(int)
}
