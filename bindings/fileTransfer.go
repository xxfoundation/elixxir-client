////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"encoding/base64"
	"encoding/json"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"time"

	"gitlab.com/elixxir/client/v4/fileTransfer"
	"gitlab.com/elixxir/client/v4/fileTransfer/e2e"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/xx_network/primitives/id"
)

////////////////////////////////////////////////////////////////////////////////
// File Transfer Structs and Interfaces                                       //
////////////////////////////////////////////////////////////////////////////////

// FileTransfer object is a bindings-layer struct which wraps a
// [fileTransfer.FileTransfer] interface.
type FileTransfer struct {
	w *e2e.Wrapper
}

// ReceivedFile contains the metadata of a new received file transfer. It is
// received from a sender on a new file transfer. It is returned by
// [ReceiveFileCallback.Callback].
//
// Example JSON:
//
//	{
//	  "TransferID": "0U+QY1nMOUzQGxGpqZyxDw8Cd6+qm8t870CzLtVoUM8=",
//	  "SenderID": "UL3+S8XdJHAfUtCUm7iZMxW8orR8Nd5JM9Ky7/5jds8D",
//	  "Preview": "aXQNcyBtZSBhIHByZXZpZXc=",
//	  "Name": "testfile.txt",
//	  "Type": "text file",
//	  "Size": 2048
//	}
type ReceivedFile struct {
	TransferID *ftCrypto.TransferID // ID of the file transfer
	SenderID   *id.ID               // ID of the file sender
	Preview    []byte               // A preview of the file
	Name       string               // Name of the file
	Type       string               // String that indicates type of file
	Size       int                  // The size of the file, in bytes
}

// FileSend contains the file and its metadata to send. This structure is JSON
// marshalled and passed as the payload to [FileTransfer.Send].
//
// Example JSON:
//
//	{
//	  "Name": "testfile.txt",
//	  "Type": "text file",
//	  "Preview": "RMlsZSBwCmV2aWV3Lg==",
//	  "Contents": "RMlsZSBjb250ZW50cy4="
//	}
type FileSend struct {
	// Name is the human-readable file name. Get max length from
	// [FileTransfer.MaxFileNameLen].
	Name string

	// Type is a shorthand that identifies the type of file. Get max length from
	// [FileTransfer.MaxFileTypeLen].
	Type string

	// Preview of the file data (e.g. a thumbnail). Get max length from
	// [FileTransfer.MaxPreviewSize].
	Preview []byte

	// Contents is the full file contents. Get max length from
	// [FileTransfer.MaxFileSize].
	Contents []byte
}

// Progress contains the progress information of a transfer. It is returned by
// [FileTransferSentProgressCallback.Callback] and
// [FileTransferReceiveProgressCallback.Callback].
//
// Example JSON:
//
//	{
//	  "TransferID": "RyJcMqtI3IIM1+YMxRwCcFiOX6AGuIzS+vQaPnqXVT8=",
//	  "Completed": false,
//	  "Transmitted": 128,
//	  "Total": 2048
//	}
type Progress struct {
	TransferID  *ftCrypto.TransferID // Transfer ID
	Completed   bool                 // Status of transfer (true if done)
	Transmitted int                  // Number of file parts sent/received
	Total       int                  // Total number of file parts
}

// ReceiveFileCallback is a bindings-layer interface that contains a callback
// that is called when a file is received.
type ReceiveFileCallback interface {
	// Callback is called when a new file transfer is received.
	//
	// Parameters:
	//  - payload - JSON of [ReceivedFile], which contains information about the
	//    incoming file transfer.
	Callback(payload []byte)
}

// FileTransferSentProgressCallback is a bindings-layer interface that contains
// a callback that is called when the sent progress updates.
type FileTransferSentProgressCallback interface {
	// Callback is called when a file part is sent or an error occurs. Once a
	// transfer completes, it should be closed using [FileTransfer.CloseSend].
	//
	// Parameters:
	//  - payload - JSON of [Progress], which describes the progress of the
	//    current transfer.
	//  - t - file part tracker that allows the lookup of the status of
	//    individual file parts.
	//  - err - Fatal errors during sending. If an error is returned, the
	//    transfer has failed and will not resume. It must be cleared using
	//    [FileTransfer.CloseSend].
	Callback(payload []byte, t *FilePartTracker, err error)
}

// FileTransferReceiveProgressCallback is a bindings-layer interface that is
// called with the progress of a received file.
type FileTransferReceiveProgressCallback interface {
	// Callback is called when a file part is received or an error occurs. Once
	// a transfer completes, the file can be received using
	// [FileTransfer.Receive].
	//
	// Parameters:
	//  - payload - JSON of [Progress], which describes the progress of the
	//    current transfer.
	//  - t - file part tracker that allows the lookup of the status of
	//    individual file parts.
	//  - err - Fatal errors during receiving. If an error is returned, the
	//    transfer has failed and will not resume.
	Callback(payload []byte, t *FilePartTracker, err error)
}

////////////////////////////////////////////////////////////////////////////////
// Main functions                                                             //
////////////////////////////////////////////////////////////////////////////////

// InitFileTransfer creates a bindings-level file transfer manager.
//
// Parameters:
//   - e2eID - ID of [E2e] object in tracker.
//   - receiveFileCallback - A callback that is called when a new file transfer
//     is received.
//   - e2eFileTransferParamsJson - JSON of
//     [gitlab.com/elixxir/client/v4/fileTransfer/e2e.Params].
//   - fileTransferParamsJson - JSON of [fileTransfer.Params].
//
// Returns:
//   - New [FileTransfer] object.
func InitFileTransfer(e2eID int, receiveFileCallback ReceiveFileCallback,
	e2eFileTransferParamsJson, fileTransferParamsJson []byte) (*FileTransfer, error) {
	jww.INFO.Printf("[FT] Calling InitFileTransfer(e2eID:%d params:%s)",
		e2eID, fileTransferParamsJson)

	// Get user from singleton
	user, err := e2eTrackerSingleton.get(e2eID)
	if err != nil {
		return nil, err
	}

	e2eFileTransferParams, err :=
		parseE2eFileTransferParams(e2eFileTransferParamsJson)
	if err != nil {
		return nil, err
	}

	fileTransferParams, err := parseFileTransferParams(fileTransferParamsJson)
	if err != nil {
		return nil, err
	}

	// Create file transfer manager
	m, err := fileTransfer.NewManager(fileTransferParams, user.api)
	if err != nil {
		return nil, errors.Errorf(
			"could not create new file transfer manager: %+v", err)
	}

	rcb := func(tid *ftCrypto.TransferID, fileName, fileType string,
		sender *id.ID, size uint32, preview []byte) {
		data, err := json.Marshal(ReceivedFile{
			TransferID: tid,
			SenderID:   sender,
			Preview:    preview,
			Name:       fileName,
			Type:       fileType,
			Size:       int(size),
		})
		if err != nil {
			jww.FATAL.Panicf(
				"[FT] Failed to JSON marshal ReceivedFile: %+v", err)
		}
		receiveFileCallback.Callback(data)
	}

	w, err := e2e.NewWrapper(rcb, e2eFileTransferParams, m, user.api)
	if err != nil {
		return nil, err
	}

	// Add file transfer processes to API services tracking
	err = user.api.AddService(m.StartProcesses)
	if err != nil {
		return nil, err
	}

	// Return wrapped manager
	return &FileTransfer{w: w}, nil
}

// Send initiates the sending of a file to a recipient and returns a transfer ID
// that uniquely identifies this file transfer. Progress for the file transfer
// is reported to that passed in callback.
//
// Parameters:
//   - payload - JSON of [FileSend], which contains the file contents and its
//     metadata.
//   - recipientID - marshalled bytes of the recipient's [id.ID].
//   - retry - The number of sending retries allowed on send failure (e.g. a
//     retry of 2.0 with 6 parts means 12 total possible sends).
//   - callback - A callback that reports the progress of the file transfer. The
//     callback is called once on initialization, on every progress update (or
//     less if restricted by the period), or on fatal error.
//   - period - The progress callback will be limited from triggering only once
//     per period. It is a duration in milliseconds. This value should depend on
//     how frequently you want to receive updates, and should be tuned to your
//     implementation.
//
// Returns:
//   - The bytes of the unique [fileTransfer.TransferID].
func (f *FileTransfer) Send(payload, recipientID []byte, retry float32,
	callback FileTransferSentProgressCallback, period int) ([]byte, error) {
	jww.INFO.Printf("[FT] Sending file transfer to %s.",
		base64.StdEncoding.EncodeToString(recipientID))

	// Unmarshal recipient ID
	recipient, err := id.Unmarshal(recipientID)
	if err != nil {
		return nil, err
	}

	p := time.Millisecond * time.Duration(period)

	// Wrap transfer progress callback to be passed to fileTransfer layer
	cb := func(completed bool, arrived, total uint16,
		st fileTransfer.SentTransfer, t fileTransfer.FilePartTracker, err error) {
		progress := &Progress{
			TransferID:  st.TransferID(),
			Completed:   completed,
			Transmitted: int(arrived),
			Total:       int(total),
		}
		pm, err2 := json.Marshal(progress)
		if err2 != nil {
			jww.FATAL.Panicf(
				"[FT] Failed to JSON marshal sent Progress object: %+v", err)
		}
		callback.Callback(pm, &FilePartTracker{t}, err)
	}

	// Unmarshal payload
	var fs FileSend
	if err = json.Unmarshal(payload, &fs); err != nil {
		return nil, err
	}

	// Send file
	ftID, err := f.w.Send(
		recipient, fs.Name, fs.Type, fs.Contents, retry, fs.Preview, cb, p)
	if err != nil {
		return nil, err
	}

	// Return Transfer ID
	return ftID.Bytes(), nil
}

// Receive returns the full file on the completion of the transfer. It deletes
// internal references to the data and unregisters any attached progress
// callback. Returns an error if the transfer is not complete, the full file
// cannot be verified, or if the transfer cannot be found.
//
// Receive can only be called once the progress callback returns that the file
// transfer is complete.
//
// Parameters:
//   - tidBytes - The file transfer's unique [fileTransfer.TransferID].
func (f *FileTransfer) Receive(tidBytes []byte) ([]byte, error) {
	tid := ftCrypto.UnmarshalTransferID(tidBytes)
	return f.w.Receive(&tid)
}

// CloseSend deletes a file from the internal storage once a transfer has
// completed or reached the retry limit. If neither of those condition are met,
// an error is returned.
//
// This function should be called once a transfer completes or errors out (as
// reported by the progress callback).
//
// Parameters:
//   - tidBytes - the file transfer's unique [fileTransfer.TransferID].
func (f *FileTransfer) CloseSend(tidBytes []byte) error {
	tid := ftCrypto.UnmarshalTransferID(tidBytes)
	return f.w.CloseSend(&tid)
}

////////////////////////////////////////////////////////////////////////////////
// Callback Registration Functions                                            //
////////////////////////////////////////////////////////////////////////////////

// RegisterSentProgressCallback allows for the registration of a callback to
// track the progress of an individual sent file transfer.
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
// Parameters:
//   - tidBytes - The file transfer's unique [fileTransfer.TransferID].
//   - callback - A callback that reports the progress of the file transfer. The
//     callback is called once on initialization, on every progress update (or
//     less if restricted by the period), or on fatal error.
//   - period - The progress callback will be limited from triggering only once
//     per period. It is a duration in milliseconds. This value should depend on
//     how frequently you want to receive updates, and should be tuned to your
//     implementation.
func (f *FileTransfer) RegisterSentProgressCallback(tidBytes []byte,
	callback FileTransferSentProgressCallback, period int) error {
	cb := func(completed bool, arrived, total uint16,
		st fileTransfer.SentTransfer, t fileTransfer.FilePartTracker, err error) {
		progress := &Progress{
			TransferID:  st.TransferID(),
			Completed:   completed,
			Transmitted: int(arrived),
			Total:       int(total),
		}
		pm, err2 := json.Marshal(progress)
		if err2 != nil {
			jww.FATAL.Panicf(
				"[FT] Failed to JSON marshal sent Progress object: %+v", err)
		}
		callback.Callback(pm, &FilePartTracker{t}, err)
	}
	p := time.Millisecond * time.Duration(period)
	tid := ftCrypto.UnmarshalTransferID(tidBytes)

	return f.w.RegisterSentProgressCallback(&tid, cb, p)
}

// RegisterReceivedProgressCallback allows for the registration of a callback to
// track the progress of an individual received file transfer.
//
// The callback will be called immediately when added to report the current
// progress of the transfer. It will then call every time a file part is
// received, the transfer completes, or a fatal error occurs. It is called at
// most once every period regardless of the number of progress updates.
//
// In the event that the client is closed and resumed, this function must be
// used to re-register any callbacks previously registered.
//
// Once the callback reports that the transfer has completed, the recipient can
// get the full file by calling Receive.
//
// Parameters:
//   - tidBytes - The file transfer's unique [fileTransfer.TransferID].
//   - callback - A callback that reports the progress of the file transfer. The
//     callback is called once on initialization, on every progress update (or
//     less if restricted by the period), or on fatal error.
//   - period - The progress callback will be limited from triggering only once
//     per period. It is a duration in milliseconds. This value should depend on
//     how frequently you want to receive updates, and should be tuned to your
//     implementation.
func (f *FileTransfer) RegisterReceivedProgressCallback(tidBytes []byte,
	callback FileTransferReceiveProgressCallback, period int) error {
	cb := func(completed bool, received, total uint16,
		rt fileTransfer.ReceivedTransfer, t fileTransfer.FilePartTracker,
		err error) {
		progress := &Progress{
			TransferID:  rt.TransferID(),
			Completed:   completed,
			Transmitted: int(received),
			Total:       int(total),
		}
		pm, err2 := json.Marshal(progress)
		if err2 != nil {
			jww.FATAL.Panicf(
				"[FT] Failed to JSON marshal received Progress object: %+v", err)
		}
		callback.Callback(pm, &FilePartTracker{t}, err)
	}
	p := time.Millisecond * time.Duration(period)

	tid := ftCrypto.UnmarshalTransferID(tidBytes)
	return f.w.RegisterReceivedProgressCallback(&tid, cb, p)
}

////////////////////////////////////////////////////////////////////////////////
// Utility Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

// MaxFileNameLen returns the max number of bytes allowed for a file name.
func (f *FileTransfer) MaxFileNameLen() int {
	return f.w.MaxFileNameLen()
}

// MaxFileTypeLen returns the max number of bytes allowed for a file type.
func (f *FileTransfer) MaxFileTypeLen() int {
	return f.w.MaxFileTypeLen()
}

// MaxFileSize returns the max number of bytes allowed for a file.
func (f *FileTransfer) MaxFileSize() int {
	return f.w.MaxFileSize()
}

// MaxPreviewSize returns the max number of bytes allowed for a file preview.
func (f *FileTransfer) MaxPreviewSize() int {
	return f.w.MaxPreviewSize()
}

////////////////////////////////////////////////////////////////////////////////
// File Part Tracker                                                          //
////////////////////////////////////////////////////////////////////////////////

// FilePartTracker contains the fileTransfer.FilePartTracker.
type FilePartTracker struct {
	m fileTransfer.FilePartTracker
}

// GetPartStatus returns the status of the file part with the given part number.
//
// The possible values for the status are:
//   - 0 < Part does not exist
//   - 0 = unsent
//   - 1 = arrived (sender has sent a part, and it has arrived)
//   - 2 = received (receiver has received a part)
func (fpt FilePartTracker) GetPartStatus(partNum int) int {
	return int(fpt.m.GetPartStatus(uint16(partNum)))
}

// GetNumParts returns the total number of file parts in the transfer.
func (fpt FilePartTracker) GetNumParts() int {
	return int(fpt.m.GetNumParts())
}

////////////////////////////////////////////////////////////////////////////////
// Event Reporter                                                             //
////////////////////////////////////////////////////////////////////////////////

// EventReport is a public struct which represents the contents of an event
// report.
//
// Example JSON:
//
//	{
//	 "Priority": 1,
//	 "Category": "Test Events",
//	 "EventType": "Ping",
//	 "Details": "This is an example of an event report"
//	}
type EventReport struct {
	Priority  int
	Category  string
	EventType string
	Details   string
}

// ReporterFunc is a bindings-layer interface that receives info from the Event
// Manager.
//
// Parameters:
//   - payload - JSON marshalled EventReport object
type ReporterFunc interface {
	Report(payload []byte, err error)
}

// reporter is the internal struct to match the event.Reporter interface.
type reporter struct {
	r ReporterFunc
}

// Report matches the event.Reporter interface, wraps the info in an EventReport
// struct, and passes the marshalled struct to the internal callback.
func (r *reporter) Report(priority int, category, evtType, details string) {
	rep := &EventReport{
		Priority:  priority,
		Category:  category,
		EventType: evtType,
		Details:   details,
	}
	r.r.Report(json.Marshal(rep))
}
