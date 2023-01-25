////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"encoding/json"

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/e2e/receive"
)

// Listener provides a callback to hear a message.
//
// An object implementing this interface can be called back when the client gets
// a message of the type that the registerer specified at registration time.
type Listener interface {
	// Hear is called to receive a message in the UI.
	//
	// Parameters:
	//  - item - JSON marshalled Message object
	Hear(item []byte)

	// Name returns a name; used for debugging.
	Name() string
}

// listener is an object internal to bindings which matches the interface
// expected by RegisterListener.
//
// It wraps the Listener type, which is usable by the bindings layer.
type listener struct {
	l Listener
}

// Message is the bindings' representation of a receive.Message.
//
// JSON example:
//  {
//   "MessageType":1,
//   "ID":"EB/70R5HYEw5htZ4Hg9ondrn3+cAc/lH2G0mjQMja3w=",
//   "Payload":"7TzZKgNphT5UooNM7mDSwtVcIs8AIu4vMKm4ld6GSR8YX5GrHirixUBAejmsgdroRJyo06TkIVef7UM9FN8YfQ==",
//   "Sender":"emV6aW1hAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD",
//   "RecipientID":"amFrZXh4MzYwAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD",
//   "EphemeralID":17,"Timestamp":1653580439357351000,
//   "Encrypted":false,
//   "RoundId":19
//  }
type Message struct {
	MessageType int
	ID          []byte
	Payload     []byte

	Sender      []byte
	RecipientID []byte
	EphemeralID int64
	Timestamp   int64 // Message timestamp of when the user sent

	Encrypted bool
	RoundId   int
	RoundURL  string
}

// Hear is called to receive a message in the UI.
func (l listener) Hear(item receive.Message) {
	m := Message{
		MessageType: int(item.MessageType),
		ID:          item.ID.Marshal(),
		Payload:     item.Payload,
		Sender:      item.Sender.Marshal(),
		RecipientID: item.RecipientID.Marshal(),
		EphemeralID: item.EphemeralID.Int64(),
		Timestamp:   item.Timestamp.UnixNano(),
		Encrypted:   item.Encrypted,
		RoundId:     int(item.Round.ID),
		RoundURL:    getRoundURL(item.Round.ID),
	}
	result, err := json.Marshal(&m)
	if err != nil {
		jww.ERROR.Printf("Unable to marshal Message: %+v", err.Error())
	}
	l.l.Hear(result)
}

// Name used for debugging.
func (l listener) Name() string {
	return l.l.Name()
}
