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

	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/client/v4/stoppable"
	"gitlab.com/elixxir/client/v4/xxdk"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/elixxir/crypto/message"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
)

// FileTransfer facilities the sending and receiving of large file transfers.
// It allows for progress tracking of both inbound and outbound transfers.
// FileTransfer handles the sending of the file data; however, the caller is
// responsible for communicating to the recipient of the incoming file transfer.
type FileTransfer interface {

	// StartProcesses starts the sending threads that wait for file transfers to
	// send. Adheres to the xxdk.Service type.
	StartProcesses() (stoppable.Stoppable, error)

	// MaxFileNameLen returns the max number of bytes allowed for a file name.
	MaxFileNameLen() int

	// MaxFileTypeLen returns the max number of bytes allowed for a file type.
	MaxFileTypeLen() int

	// MaxFileSize returns the max number of bytes allowed for a file.
	MaxFileSize() int

	// MaxPreviewSize returns the max number of bytes allowed for a file
	// preview.
	MaxPreviewSize() int

	/* === Sending ========================================================== */
	// The process of sending a file involves three main steps:
	//  1. Upload the file to a new identity using Upload. It is added to the
	//     event model with the status Uploading.
	//  2. Wait for the file to be marked as Complete in the event model and get
	//     the file info stored there.
	//  3. Send the file info to the channel using Send.
	//
	// When the file is uploaded, it is broken into individual, equal-length
	// parts and sent to a randomly generated ID. Every time one of these parts
	// is sent or received, it is reported on all registered
	// SentProgressCallbacks for that transfer.
	//
	// A SentProgressCallback is registered on the initial upload. However, if
	// the client is closed and reopened, the callback must be registered again
	// using RegisterSentProgressCallback; otherwise, the continued progress of
	// the transfer will not be reported.

	// Upload starts uploading the file to a new ID that can be sent to the
	// specified channel when complete. To get progress information about the
	// upload, a SentProgressCallback must be registered. All errors returned on
	// the callback are fatal and the user must take action to either
	// RetryUpload or CloseSend.
	//
	// The file is added to the event model at the returned file ID with the
	// status Uploading. Once the upload is complete, the file link is added to
	// the event model with the status Complete.
	//
	// The SentProgressCallback only indicates the progress of the file upload,
	// not the status of the file in the event model. You must rely on updates
	// from the event model to know when it can be retrieved.
	//
	// Parameters:
	//   - fileData - File contents. Max size defined by MaxFileSize.
	//   - retry - The number of sending retries allowed on send failure (e.g.
	//     a retry of 2.0 with 6 parts means 12 total possible sends).
	//   - progressCB - A callback that reports the progress of the file upload.
	//     The callback is called once on initialization, on every progress
	//     update (or less if restricted by the period), or on fatal error.
	//   - period - A progress callback will be limited from triggering only
	//     once per period.
	//
	// Returns:
	//   - A file ID that uniquely identifies this file.
	Upload(fileData []byte, retry float32, progressCB SentProgressCallback,
		period time.Duration) (ftCrypto.ID, error)

	// Send sends the specified file info to the channel. Once a file is
	// uploaded via Upload, its file link (found in the event model) can be sent
	// to any channel.
	//
	// Parameters:
	//   - channelID - The ID of the channel to send the file to.
	//   - fileLink - JSON of FileLink stored in the event model.
	//   - fileName - Human-readable file name. Max length defined by
	//     MaxFileNameLen.
	//   - fileType - Shorthand that identifies the type of file. Max length
	//     defined by MaxFileTypeLen.
	//   - preview - A preview of the file data (e.g. a thumbnail). Max size
	//     defined by MaxPreviewSize.
	//   - validUntil - The duration that the file is available in the channel.
	//     For the maximum amount of time, use channels.ValidForever.
	//   - params - The xxdk.CMIXParams to send this.
	Send(channelID *id.ID, fileLink []byte, fileName, fileType string,
		preview []byte, validUntil time.Duration, params xxdk.CMIXParams) (
		message.ID, rounds.Round, ephemeral.Id, error)

	// RegisterSentProgressCallback allows for the registration of a callback to
	// track the progress of an individual file upload. A SentProgressCallback
	// is auto-registered on Send; this function should be called when resuming
	// clients or registering extra callbacks.
	//
	// The callback will be called immediately when added to report the current
	// progress of the transfer. It will then call every time a file part
	// arrives, the transfer completes, or a fatal error occurs. It is called at
	// most once every period regardless of the number of progress updates.
	//
	// In the event that the client is closed and resumed, this function must be
	// used to re-register any callbacks previously registered with this
	// function or Send.
	//
	// The SentProgressCallback only indicates the progress of the file upload,
	// not the status of the file in the event model. You must rely on updates
	// from the event model to know when it can be retrieved.
	//
	// Parameters:
	//   - fileID - The unique ID of the file.
	//   - progressCB - A callback that reports the progress of the file upload.
	//     The callback is called once on initialization, on every progress
	//     update (or less if restricted by the period), or on fatal error.
	//   - period - A progress callback will be limited from triggering only
	//     once per period.
	RegisterSentProgressCallback(fileID ftCrypto.ID,
		progressCB SentProgressCallback, period time.Duration) error

	// RetryUpload retries uploading a failed file upload. Returns an error if
	// the transfer has not failed.
	//
	// This function should be called once a transfer errors out (as reported by
	// the progress callback).
	//
	// A new progress callback must be registered on retry. Any previously
	// registered callbacks are defunct when the upload fails.
	RetryUpload(fileID ftCrypto.ID,
		progressCB SentProgressCallback, period time.Duration) error

	// CloseSend deletes a file from the internal storage once a transfer has
	// completed or reached the retry limit. If neither of those condition are
	// met, an error is returned.
	//
	// This function should be called once a transfer completes or errors out
	// (as reported by the progress callback).
	CloseSend(fileID ftCrypto.ID) error

	/* === Receiving ======================================================== */
	// The process of receiving a file involves four main steps:
	//  1. Receive file info from a channel message in the event model.
	//  2. Initiate download and register a progress callback with Download. It
	//     will be added to the event model with the status Downloading.
	//  3. Receive transfer progress on the progress callback.
	//  4. Once the status of the file is marked Complete in the event model, it
	//     can be downloaded.
	//
	// All file downloads are initiated by the user from file links they
	// receive. A ReceivedProgressCallback must be registered to see the
	// download progress. But the file can only be downloaded once marked
	// Complete in the event model.
	//
	// A ReceivedProgressCallback is registered on the initial download.
	// However, if the client is closed and reopened, the callback must be
	// registered again using RegisterReceivedProgressCallback; otherwise, the
	// continued progress of the download will not be reported.

	// Download begins the download of the file described in the marshalled
	// FileInfo. The progress of the download is reported on the
	// ReceivedProgressCallback.
	//
	// Once the download completes, the file will be stored in the event model
	// with the given file ID and with the status
	// channels.ReceptionProcessingComplete.
	//
	// The ReceivedProgressCallback only indicates the progress of the file
	// download, not the status of the file in the event model. You must rely on
	// updates from the event model to know when it can be retrieved.
	//
	// Parameters:
	//   - fileInfo - The JSON of FileInfo received on a channel.
	//   - progressCB - A callback that reports the progress of the file
	//     download. The callback is called once on initialization, on every
	//     progress update (or less if restricted by the period), or on fatal
	//     error.
	//   - period - A progress callback will be limited from triggering only
	//     once per period.
	//
	// Returns:
	//   - A file ID that uniquely identifies this file.
	Download(fileInfo []byte, progressCB ReceivedProgressCallback,
		period time.Duration) (ftCrypto.ID, error)

	// RegisterReceivedProgressCallback allows for the registration of a
	// callback to track the progress of an individual file download.
	//
	// The callback will be called immediately when added to report the current
	// progress of the transfer. It will then call every time a file part is
	// received, the transfer completes, or a fatal error occurs. It is called
	// at most once every period regardless of the number of progress updates.
	//
	// In the event that the client is closed and resumed, this function must be
	// used to re-register any callbacks previously registered.
	//
	// Once the download completes, the file will be stored in the event model
	// with the given file ID and with the status Complete.
	//
	// The ReceivedProgressCallback only indicates the progress of the file
	// download, not the status of the file in the event model. You must rely on
	// updates from the event model to know when it can be retrieved.
	//
	// Parameters:
	//   - fileID - The unique ID of the file.
	//   - progressCB - A callback that reports the progress of the file upload.
	//     The callback is called once on initialization, on every progress
	//     update (or less if restricted by the period), or on fatal error.
	//   - period - A progress callback will be limited from triggering only
	//     once per period.
	RegisterReceivedProgressCallback(fileID ftCrypto.ID,
		progressCB ReceivedProgressCallback, period time.Duration) error
}

// SentProgressCallback is called when the progress on a sent file changes or an
// error occurs in the transfer.
//
// The [FilePartTracker] can be used to look up the status of individual file
// parts. Note, when completed == true, the [FilePartTracker] may be nil.
//
// Any error returned is fatal and the file must either be retried with
// [FileTransfer.RetryUpload] or canceled with [FileTransfer.CloseSend].
//
// This callback only indicates the status of the file transfer, not the status
// of the file in the event model. Do NOT use this callback as an indicator of
// when the file is available in the event model.
type SentProgressCallback func(completed bool, sent, received, total uint16,
	st SentTransfer, fpt FilePartTracker, err error)

// ReceivedProgressCallback is called when the progress on a received file
// changes or an error occurs in the transfer.
//
// The [FilePartTracker] can be used to look up the status of individual file
// parts. Note, when completed == true, the [FilePartTracker] may be nil.
//
// This callback only indicates the status of the file transfer, not the status
// of the file in the event model. Do NOT use this callback as an indicator of
// when the file is available in the event model.
type ReceivedProgressCallback func(completed bool, received, total uint16,
	rt ReceivedTransfer, fpt FilePartTracker, err error)

// SentTransfer tracks the information and individual parts of a sent file
// transfer.
type SentTransfer interface {
	GetRecipient() *id.ID
	Transfer
}

// ReceivedTransfer tracks the information and individual parts of a received
// file transfer.
type ReceivedTransfer interface {
	Transfer
}

// Transfer is the generic structure for a file transfer.
type Transfer interface {
	GetFileID() ftCrypto.ID
	GetFileSize() uint32
	GetNumParts() uint16
}

// FilePartTracker tracks the status of each file part in a sent or received
// file transfer.
type FilePartTracker interface {
	// GetPartStatus returns the status of the file part with the given part
	// number. The possible values for the status are:
	//  0 < Part does not exist
	//  0 = unsent
	//  1 = arrived (sender has sent a part, and it has arrived)
	//  2 = received (receiver has received a part)
	GetPartStatus(partNum uint16) FpStatus

	// GetNumParts returns the total number of file parts in the transfer.
	GetNumParts() uint16
}

// FpStatus is the file part status and indicates the status of individual file
// parts in a file transfer.
type FpStatus int

// Possible values for FpStatus.
const (
	// FpUnsent indicates that the file part has not been sent
	FpUnsent FpStatus = iota

	// FpSent indicates that the file part has been sent (sender has sent a
	// part, but it has not arrived)
	FpSent

	// FpArrived indicates that the file part has arrived (sender has sent a
	// part, and it has arrived)
	FpArrived

	// FpReceived indicates that the file part has been received (receiver has
	// received a part)
	FpReceived
)

// String returns the string representing of the FpStatus. This functions
// satisfies the fmt.Stringer interface.
func (fps FpStatus) String() string {
	switch fps {
	case FpUnsent:
		return "unsent"
	case FpSent:
		return "sent"
	case FpArrived:
		return "arrived"
	case FpReceived:
		return "received"
	default:
		return "INVALID FpStatus: " + strconv.Itoa(int(fps))
	}
}
