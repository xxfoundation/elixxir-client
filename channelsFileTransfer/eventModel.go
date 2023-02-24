////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package channelsFileTransfer

import (
	"crypto/ed25519"
	"time"

	"gitlab.com/elixxir/client/v4/channels"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/xx_network/primitives/id"
)

// EventModel is an interface that allows the user to get channels messages and
// file transfers.
type EventModel interface {
	// ReceiveFileMessage is called when a file upload begins or when a message
	// to download a file is received.
	//
	// The API needs to return a UUID of the message that can be referenced at a
	// later time.
	//
	// fileInfo, timestamp, lease, and round are nillable and may be updated
	// based upon the UUID or file ID later. A time of time.Time{} will be
	// passed for a nilled timestamp.
	//
	// nickname may be empty, in which case the UI is expected to display the
	// codename.
	ReceiveFileMessage(channelID *id.ID, fileID ftCrypto.ID, nickname string,
		fileInfo, fileData []byte, pubKey ed25519.PublicKey, dmToken uint32,
		codeset uint8, timestamp time.Time, lease time.Duration,
		round rounds.Round, messageType channels.MessageType,
		status channels.SentStatus, hidden bool) uint64

	// UpdateFile is called when a file upload completed, a download starts, or
	// a download completes. Each use will be identified in a SentStatus change
	// (SendProcessingComplete, ReceptionProcessing, and
	// ReceptionProcessingComplete).
	//
	// timestamp, round, pinned, hidden, and status are all nillable and may be
	// updated based upon the fileID at a later date. If a nil value is passed,
	// then make no update.
	//
	// Returns an error if the message cannot be updated. It must return
	// channels.NoMessageErr if the message does not exist.
	UpdateFile(fileID ftCrypto.ID, fileInfo, fileData *[]byte,
		timestamp *time.Time, round *rounds.Round, pinned, hidden *bool,
		status *channels.SentStatus) error

	// GetFile returns the file data and info at the given file ID.
	//
	// Returns an error if the file cannot be gotten. It must return
	// channels.NoMessageErr if the file does not exist.
	GetFile(fileID ftCrypto.ID) (fileInfo, fileData []byte, err error)

	// DeleteFile deletes the file with the given file ID.
	//
	// It must return channels.NoMessageErr if the file does not exist.
	DeleteFile(fileID ftCrypto.ID) error

	channels.EventModel
}
