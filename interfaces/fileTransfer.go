////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package interfaces

import (
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

// SentProgressCallback is a callback function that tracks the progress of
// sending a file.
type SentProgressCallback func(completed bool, sent, arrived, total uint16,
	err error)

// ReceivedProgressCallback is a callback function that tracks the progress of
// receiving a file.
type ReceivedProgressCallback func(completed bool, received, total uint16,
	err error)

// ReceiveCallback is a callback function that notifies the receiver of an
// incoming file transfer.
type ReceiveCallback func(tid ftCrypto.TransferID, fileName, fileType string,
	sender *id.ID, size uint32, preview []byte)

// FileTransfer facilities the sending and receiving of large file transfers.
// It allows for progress tracking of both inbound and outbound transfers.
type FileTransfer interface {
	// Send sends a file to the recipient. The sender must have an E2E
	// relationship with the recipient.
	// The retry float is the total amount of data to send relative to the data
	// size. Data will be resent on error and will resend up to [(1 + retry) *
	// fileSize].
	// The preview stores a preview of the data (such as a thumbnail) and is
	// capped at 4 kB in size.
	// Returns a unique transfer ID used to identify the transfer.
	Send(fileName, fileType string, fileData []byte, recipient *id.ID,
		retry float32, preview []byte, progressCB SentProgressCallback,
		period time.Duration) (ftCrypto.TransferID, error)

	// RegisterSendProgressCallback allows for the registration of a callback to
	// track the progress of an individual sent file transfer. The callback will
	// be called immediately when added to report the current status of the
	// transfer. It will then call every time a file part is sent, a file part
	// arrives, the transfer completes, or an error occurs. It is called at most
	// once ever period, which means if events occur faster than the period,
	// then they will not be reported and instead the progress will be reported
	// once at the end of the period.
	RegisterSendProgressCallback(tid ftCrypto.TransferID,
		progressCB SentProgressCallback, period time.Duration) error

	// Resend resends a file if sending fails. Returns an error if CloseSend
	// was already called or if the transfer did not run out of retries.
	Resend(tid ftCrypto.TransferID) error

	// CloseSend deletes a file from the internal storage once a transfer has
	// completed or reached the retry limit. Returns an error if the transfer
	// has not run out of retries.
	CloseSend(tid ftCrypto.TransferID) error

	// RegisterReceiveProgressCallback allows for the registration of a callback
	// to track the progress of an individual received file transfer. The
	// callback will be called immediately when added to report the current
	// status of the transfer. It will then call every time a file part is
	// received, the transfer completes, or an error occurs. It is called at
	// most once ever period, which means if events occur faster than the
	// period, then they will not be reported and instead the progress will be
	// reported once at the end of the period.
	// Once the callback reports that the transfer has completed, the recipient
	// can get the full file by calling Receive.
	RegisterReceiveProgressCallback(tid ftCrypto.TransferID,
		progressCB ReceivedProgressCallback, period time.Duration) error

	// Receive returns the full file on the completion of the transfer as
	// reported by a registered ReceivedProgressCallback. It deletes internal
	// references to the data and unregisters any attached progress callback.
	// Returns an error if the transfer is not complete, the full file cannot be
	// verified, or if the transfer cannot be found.
	Receive(tid ftCrypto.TransferID) ([]byte, error)
}
