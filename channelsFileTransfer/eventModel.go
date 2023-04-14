////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package channelsFileTransfer

import (
	"strconv"
	"time"

	"gitlab.com/elixxir/client/v4/channels"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
)

// EventModel is an interface that allows the user to get channels messages and
// file transfers.
type EventModel interface {
	// ReceiveFile is called when a file upload or download beings.
	//
	// fileLink and fileData are nillable and may be updated based upon the UUID
	// or file ID later.
	//
	// fileID is always unique to the fileData. fileLink is the JSON of
	// FileLink.
	//
	// Returns any fatal errors.
	ReceiveFile(fileID ftCrypto.ID, fileLink, fileData []byte,
		timestamp time.Time, status Status) error

	// UpdateFile is called when a file upload or download completes or changes.
	//
	// fileLink, fileData, timestamp, and status are all nillable and may be
	// updated based upon the file ID at a later date. If a nil value is passed,
	// then make no update.
	//
	// Returns an error if the file cannot be updated. It must return
	// channels.NoMessageErr if the file does not exist.
	UpdateFile(fileID ftCrypto.ID, fileLink, fileData []byte,
		timestamp *time.Time, status *Status) error

	// GetFile returns the ModelFile containing the file data and download link
	// for the given file ID.
	//
	// Returns an error if the file cannot be retrieved. It must return
	// channels.NoMessageErr if the file does not exist.
	GetFile(fileID ftCrypto.ID) (ModelFile, error)

	// DeleteFile deletes the file with the given file ID.
	//
	// Returns fatal errors. It must return channels.NoMessageErr if the file
	// does not exist.
	DeleteFile(fileID ftCrypto.ID) error

	channels.EventModel
}

// ModelFile contains a file and all of its information.
type ModelFile struct {
	// ID is the unique identifier of this file.
	ID ftCrypto.ID `json:"id"`

	// Link contains all the information needed to download the file data.
	// It is the JSON of [FileLink].
	Link []byte `json:"link"`

	// Data is the contents of the file.
	Data []byte `json:"fileData"`

	// Timestamp is the last time the file data, link, or status was modified.
	Timestamp time.Time `json:"timestamp"`

	// The current status of the file in the event model.
	Status Status `json:"status"`
}

// Status is the current status of a file stored in the event model.
type Status uint8

const (
	// NotStarted indicates that the file has been added to the file transfer
	// manager, but it has yet to start uploading or downloading.
	// NotStarted Status = 0

	// Uploading indicates that the file is currently being uploaded. In this
	// state, the file data is accessible but the file link is not.
	Uploading Status = 10

	// Downloading indicates that the file is currently being downloaded. In
	// this state, the file link is accessible but the file data is not.
	Downloading Status = 20

	// Complete indicates that the file has successfully finished uploading or
	// downloading and the file is available to send/receive. In this state,
	// both the file data and file link are accessible.
	Complete Status = 30

	// Error indicates a fatal error occurred during upload or download.
	Error Status = 40
)

// String returns the human-readable form of the [Status] for logging and
// debugging. This function adheres to the [fmt.Stringer] interface.
func (ft Status) String() string {
	switch ft {
	// case NotStarted:
	// 	return "not started"
	case Uploading:
		return "uploading"
	case Downloading:
		return "downloading"
	case Complete:
		return "complete"
	case Error:
		return "error"
	default:
		return "INVALID STATUS: " + strconv.Itoa(int(ft))
	}
}
