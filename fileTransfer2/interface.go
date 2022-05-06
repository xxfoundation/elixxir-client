////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package fileTransfer2

import (
	"gitlab.com/elixxir/client/stoppable"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/xx_network/primitives/id"
	"strconv"
	"time"
)

// SentProgressCallback is a callback function that tracks the progress of
// sending a file.
type SentProgressCallback func(completed bool, arrived, total uint16,
	st SentTransfer, t FilePartTracker, err error)

// ReceivedProgressCallback is a callback function that tracks the progress of
// receiving a file.
type ReceivedProgressCallback func(completed bool, received, total uint16,
	rt ReceivedTransfer, t FilePartTracker, err error)

// ReceiveCallback is a callback function that notifies the receiver of an
// incoming file transfer.
type ReceiveCallback func(tid *ftCrypto.TransferID, fileName, fileType string,
	sender *id.ID, size uint32, preview []byte)

// SendNew handles the sending of the initial message informing the recipient
// of the incoming file transfer parts. SendNew should block until the send
// completes and return an error only on failed sends.
type SendNew func(transferInfo []byte) error

// FileTransfer facilities the sending and receiving of large file transfers.
// It allows for progress tracking of both inbound and outbound transfers.
// FileTransfer handles the sending of the file data; however, the caller is
// responsible for communicating to the recipient of the incoming file transfer.
type FileTransfer interface {

	// StartProcesses starts the sending threads that wait for file transfers to
	// send. Adheres to the api.Service type.
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
	/* The processes of sending a file involves four main steps:
		 1. Set up a method to send initial file transfer details using SendNew.
		 2. Sending the file using Send and register a progress callback.
		 3. Receiving transfer progress on the progress callback.
	     4. Closing a finished send using CloseSend.

	   Once the file is sent, it is broken into individual, equal-length parts
	   and sent to the recipient. Every time one of these parts arrives, it is
	   reported on all registered SentProgressCallbacks for that transfer.

	   A SentProgressCallback is registered on the initial send. However, if the
	   client is closed and reopened, the callback must be registered again
	   using RegisterSentProgressCallback, otherwise the continued progress of
	   the transfer will not be reported.

	   Once the SentProgressCallback returns that the file has completed
	   sending, the file can be closed using CloseSend. If the callback reports
	   an error, then the file should also be closed using CloseSend.
	*/

	// Send initiates the sending of a file to the recipient and returns a
	// transfer ID that uniquely identifies this file transfer.
	//
	// In-progress transfers are restored when closing and reopening; however, a
	// SentProgressCallback must be registered again.
	//
	//   recipient - ID of the receiver of the file transfer. The sender must
	//      have an E2E relationship with the recipient.
	//   fileName - Human-readable file name. Max length defined by
	//      MaxFileNameLen.
	//   fileType - Shorthand that identifies the type of file. Max length
	//      defined by MaxFileTypeLen.
	//   fileData - File contents. Max size defined by MaxFileSize.
	//   retry - The number of sending retries allowed on send failure (e.g.
	//      a retry of 2.0 with 6 parts means 12 total possible sends).
	//   preview - A preview of the file data (e.g. a thumbnail). Max size
	//      defined by MaxPreviewSize.
	//   progressCB - A callback that reports the progress of the file transfer.
	//      The callback is called once on initialization, on every progress
	//      update (or less if restricted by the period), or on fatal error.
	//   period - A progress callback will be limited from triggering only once
	//      per period.
	//   sendNew - Function that sends the file transfer information to the
	//      recipient.
	Send(recipient *id.ID, fileName, fileType string, fileData []byte,
		retry float32, preview []byte, progressCB SentProgressCallback,
		period time.Duration, sendNew SendNew) (*ftCrypto.TransferID, error)

	// RegisterSentProgressCallback allows for the registration of a callback to
	// track the progress of an individual sent file transfer.
	// SentProgressCallback is auto registered on Send; this function should be
	// called when resuming clients or registering extra callbacks.
	//
	// The callback will be called immediately when added to report the current
	// progress of the transfer. It will then call every time a file part
	// arrives, the transfer completes, or a fatal error occurs. It is called at
	// most once every period regardless of the number of progress updates.
	//
	// In the event that the client is closed and resumed, this function must be
	// used to re-register any callbacks previously registered with this
	// function or Send.
	RegisterSentProgressCallback(tid *ftCrypto.TransferID,
		progressCB SentProgressCallback, period time.Duration) error

	// CloseSend deletes a file from the internal storage once a transfer has
	// completed or reached the retry limit. Returns an error if the transfer
	// has not run out of retries.
	//
	// This function should be called once a transfer completes or errors out
	// (as reported by the progress callback).
	CloseSend(tid *ftCrypto.TransferID) error

	/* === Receiving ======================================================== */
	/* The processes of receiving a file involves four main steps:
		 1. Receiving a new file transfer through a channel set up by the
	        caller.
	     2. Registering the file transfer and a progress callback with
	        HandleIncomingTransfer.
		 3. Receiving transfer progress on the progress callback.
	     4. Receiving the complete file using Receive once the callback says
	        the transfer is complete.

	   Once the file transfer manager has started, it will call the
	   ReceiveCallback for every new file transfer that is received. Once that
	   happens, a ReceivedProgressCallback must be registered using
	   RegisterReceivedProgressCallback to get progress updates on the transfer.

	   When the progress callback reports that the transfer is complete, the
	   full file can be retrieved using Receive.
	*/

	// HandleIncomingTransfer starts tracking the received file parts for the
	// given payload that contains the file transfer information and returns a
	// transfer ID that uniquely identifies this file transfer along with the
	// transfer information
	//
	// This function should be called once for every new file received on the
	// registered SendNew callback.
	//
	// In-progress transfers are restored when closing and reopening; however, a
	// ReceivedProgressCallback must be registered again.
	//
	//   payload - A marshalled payload container the file transfer information.
	//   progressCB - A callback that reports the progress of the file transfer.
	//      The callback is called once on initialization, on every progress
	//      update (or less if restricted by the period), or on fatal error.
	//   period - A progress callback will be limited from triggering only once
	//      per period.
	HandleIncomingTransfer(transferInfo []byte,
		progressCB ReceivedProgressCallback, period time.Duration) (
		*ftCrypto.TransferID, *TransferInfo, error)

	// RegisterReceivedProgressCallback allows for the registration of a
	// callback to track the progress of an individual received file transfer.
	// This should be done when a new transfer is received on the
	// ReceiveCallback.
	//
	// The callback will be called immediately when added to report the current
	// progress of the transfer. It will then call every time a file part is
	// received, the transfer completes, or a fatal error occurs. It is called
	// at most once every period regardless of the number of progress updates.
	//
	// In the event that the client is closed and resumed, this function must be
	// used to re-register any callbacks previously registered.
	//
	// Once the callback reports that the transfer has completed, the recipient
	// can get the full file by calling Receive.
	RegisterReceivedProgressCallback(tid *ftCrypto.TransferID,
		progressCB ReceivedProgressCallback, period time.Duration) error

	// Receive returns the full file on the completion of the transfer.
	// It deletes internal references to the data and unregisters any attached
	// progress callback. Returns an error if the transfer is not complete, the
	// full file cannot be verified, or if the transfer cannot be found.
	//
	// Receive can only be called once the progress callback returns that the
	// file transfer is complete.
	Receive(tid *ftCrypto.TransferID) ([]byte, error)
}

// SentTransfer tracks the information and individual parts of a sent file
// transfer.
type SentTransfer interface {
	Recipient() *id.ID
	Transfer
}

// ReceivedTransfer tracks the information and individual parts of a received
// file transfer.
type ReceivedTransfer interface {
	Transfer
}

// Transfer is the generic structure for a file transfer.
type Transfer interface {
	TransferID() *ftCrypto.TransferID
	FileName() string
	FileSize() uint32
	NumParts() uint16
}

// FilePartTracker tracks the status of each file part in a sent or received
// file transfer.
type FilePartTracker interface {
	// GetPartStatus returns the status of the file part with the given part
	// number. The possible values for the status are:
	// 0 < Part does not exist
	// 0 = unsent
	// 1 = arrived (sender has sent a part, and it has arrived)
	// 2 = received (receiver has received a part)
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
