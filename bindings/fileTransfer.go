///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"encoding/json"
	"time"

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/fileTransfer"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/xx_network/primitives/id"
)

////////////////////////////////////////////////////////////////////////////////
// File Transfer Structs and Interfaces                                       //
////////////////////////////////////////////////////////////////////////////////

// FileTransfer object is a bindings-layer struct which wraps a
// fileTransfer.FileTransfer interface.
type FileTransfer struct {
	ft    fileTransfer.FileTransfer
	e2eCl *E2e
}

// ReceivedFile is a public struct that contains the metadata of a new file
// transfer.
//
// Example JSON:
//  {
//   "TransferID":"B4Z9cwU18beRoGbk5xBjbcd5Ryi9ZUFA2UBvi8FOHWo=",
//   "SenderID":"emV6aW1hAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD",
//   "Preview":"aXQncyBtZSBhIHByZXZpZXc=",
//   "Name":"testfile.txt",
//   "Type":"text file",
//   "Size":2048
//  }
type ReceivedFile struct {
	TransferID []byte // ID of the file transfer
	SenderID   []byte // ID of the file sender
	Preview    []byte // A preview of the file
	Name       string // Name of the file
	Type       string // String that indicates type of file
	Size       int    // The size of the file, in bytes
}

// FileSend is a public struct that contains the file contents and its name,
// type, and preview.
//  {
//   "Name":"testfile.txt",
//   "Type":"text file",
//   "Preview":"aXQncyBtZSBhIHByZXZpZXc=",
//   "Contents":"VGhpcyBpcyB0aGUgZnVsbCBjb250ZW50cyBvZiB0aGUgZmlsZSBpbiBieXRlcw=="
//  }
type FileSend struct {
	Name     string // Name of the file
	Type     string // String that indicates type of file
	Preview  []byte // A preview of the file
	Contents []byte // Full contents of the file
}

// Progress is a public struct that represents the progress of an in-progress
// file transfer.
//
// Example JSON:
//  {
//   "Completed":false,
//   "Transmitted":128,
//   "Total":2048,
//   "Err":null
//  }
type Progress struct {
	Completed   bool  // Status of transfer (true if done)
	Transmitted int   // Number of file parts sent/received
	Total       int   // Total number of file parts
	Err         error // Error status (if any)
}

// ReceiveFileCallback is a bindings-layer interface that contains a callback
// that is called when a file is received.
type ReceiveFileCallback interface {
	// Callback is called when a new file transfer is received.
	//
	// Parameters:
	//  - payload - the JSON marshalled bytes of a ReceivedFile object.
	//  - err - any errors that occurred during reception
	Callback(payload []byte, err error)
}

// FileTransferSentProgressCallback is a bindings-layer interface that contains
// a callback that is called when the sent progress updates.
type FileTransferSentProgressCallback interface {
	// Callback is called when a file part is sent or an error occurs.
	//
	// Parameters:
	//  - payload - the JSON marshalled bytes of a Progress object.
	//  - t - tracker that allows the lookup of the status of any file part
	//  - err - any errors that occurred during sending
	Callback(payload []byte, t *FilePartTracker, err error)
}

// FileTransferReceiveProgressCallback is a bindings-layer interface that is
// called with the progress of a received file.
//
type FileTransferReceiveProgressCallback interface {
	// Callback is called when a file part is sent or an error occurs.
	//
	// Parameters:
	//  - payload - the JSON marshalled bytes of a Progress object.
	//  - t - tracker that allows the lookup of the status of any file part
	//  - err - any errors that occurred during sending
	Callback(payload []byte, t *FilePartTracker, err error)
}

////////////////////////////////////////////////////////////////////////////////
// Main functions                                                             //
////////////////////////////////////////////////////////////////////////////////

// InitFileTransfer creates a bindings-level file transfer manager.
//
// Parameters:
//  - e2eID - e2e object ID in the tracker
//  - paramsJSON - JSON marshalled fileTransfer.Params
func InitFileTransfer(e2eID int, paramsJSON []byte) (*FileTransfer, error) {

	// Get user from singleton
	user, err := e2eTrackerSingleton.get(e2eID)
	if err != nil {
		return nil, err
	}

	// Client info
	myID := user.api.GetReceptionIdentity().ID
	rng := user.api.GetRng()

	params, err := parseFileTransferParams(paramsJSON)
	if err != nil {
		return nil, err
	}

	// Create file transfer manager
	m, err := fileTransfer.NewManager(params, myID,
		user.api.GetCmix(), user.api.GetStorage(), rng)

	// Add file transfer processes to client services tracking
	err = user.api.AddService(m.StartProcesses)
	if err != nil {
		return nil, err
	}

	// Return wrapped manager
	return &FileTransfer{ft: m, e2eCl: user}, nil
}

// Send is the bindings-level function for sending a file.
//
// Parameters:
//  - payload - JSON marshalled FileSend
//  - recipientID - marshalled recipient id.ID
//  - paramsJSON - JSON marshalled e2e.Params
//  - retry - number of retries allowed
//  - callback - callback that reports file sending progress
//  - period - duration to wait between progress callbacks triggering
//
// Returns:
//  - []byte - unique file transfer ID
func (f *FileTransfer) Send(payload, recipientID, paramsJSON []byte, retry float32,
	callback FileTransferSentProgressCallback, period string) ([]byte, error) {
	// Unmarshal recipient ID
	recipient, err := id.Unmarshal(recipientID)
	if err != nil {
		return nil, err
	}

	// Parse duration to time.Duration
	p, err := time.ParseDuration(period)

	// Wrap transfer progress callback to be passed to fileTransfer layer
	cb := func(completed bool, arrived, total uint16,
		st fileTransfer.SentTransfer, t fileTransfer.FilePartTracker, err error) {
		prog := &Progress{
			Completed:   completed,
			Transmitted: int(arrived),
			Total:       int(total),
			Err:         err,
		}
		pm, err := json.Marshal(prog)
		callback.Callback(pm, &FilePartTracker{t}, err)
	}

	// Unmarshal payload
	fs := &FileSend{}
	err = json.Unmarshal(payload, fs)
	if err != nil {
		return nil, err
	}

	sendNew := func(transferInfo []byte) error {
		resp, err := f.e2eCl.SendE2E(int(catalog.NewFileTransfer), recipientID, transferInfo, paramsJSON)
		if err != nil {
			return err
		}
		jww.INFO.Printf("New file transfer message sent: %s", resp)
		return nil
	}

	// Send file
	ftID, err := f.ft.Send(recipient, fs.Name, fs.Type, fs.Contents, retry, fs.Preview, cb, p, sendNew)
	if err != nil {
		return nil, err
	}

	// Return Transfer ID
	return ftID.Bytes(), nil
}

// Receive returns the full file on the completion of the transfer. It deletes
// internal references to the data and unregisters any attached progress
// callbacks. Returns an error if the transfer is not complete, the full file
// cannot be verified, or if the transfer cannot be found.
//
// Receive can only be called once the progress callback returns that the
// file transfer is complete.
//
// Parameters:
//  - tidBytes - file transfer ID
func (f *FileTransfer) Receive(tidBytes []byte) ([]byte, error) {
	tid := ftCrypto.UnmarshalTransferID(tidBytes)
	return f.ft.Receive(&tid)
}

// CloseSend deletes a file from the internal storage once a transfer has
// completed or reached the retry limit. Returns an error if the transfer has
// not run out of retries.
//
// This function should be called once a transfer completes or errors out (as
// reported by the progress callback).
//
// Parameters:
//  - tidBytes - file transfer ID
func (f *FileTransfer) CloseSend(tidBytes []byte) error {
	tid := ftCrypto.UnmarshalTransferID(tidBytes)
	return f.ft.CloseSend(&tid)
}

////////////////////////////////////////////////////////////////////////////////
// Callback Registration Functions                                            //
////////////////////////////////////////////////////////////////////////////////

// RegisterSentProgressCallback allows for the registration of a callback to
// track the progress of an individual sent file transfer.
//
// SentProgressCallback is auto registered on Send; this function should be
// called when resuming clients or registering extra callbacks.
//
// Parameters:
//  - tidBytes - file transfer ID
//  - callback - callback that reports file reception progress
//  - period - duration to wait between progress callbacks triggering
func (f *FileTransfer) RegisterSentProgressCallback(tidBytes []byte,
	callback FileTransferSentProgressCallback, period string) error {
	cb := func(completed bool, arrived, total uint16,
		st fileTransfer.SentTransfer, t fileTransfer.FilePartTracker, err error) {
		prog := &Progress{
			Completed:   completed,
			Transmitted: int(arrived),
			Total:       int(total),
			Err:         err,
		}
		pm, err := json.Marshal(prog)
		callback.Callback(pm, &FilePartTracker{t}, err)
	}
	p, err := time.ParseDuration(period)
	if err != nil {
		return err
	}
	tid := ftCrypto.UnmarshalTransferID(tidBytes)

	return f.ft.RegisterSentProgressCallback(&tid, cb, p)
}

// RegisterReceivedProgressCallback allows for the registration of a callback to
// track the progress of an individual received file transfer.
//
// This should be done when a new transfer is received on the ReceiveCallback.
//
// Parameters:
//  - tidBytes - file transfer ID
//  - callback - callback that reports file reception progress
//  - period - duration to wait between progress callbacks triggering
func (f *FileTransfer) RegisterReceivedProgressCallback(tidBytes []byte,
	callback FileTransferReceiveProgressCallback, period string) error {
	cb := func(completed bool, received, total uint16,
		rt fileTransfer.ReceivedTransfer, t fileTransfer.FilePartTracker, err error) {
		prog := &Progress{
			Completed:   completed,
			Transmitted: int(received),
			Total:       int(total),
			Err:         err,
		}
		pm, err := json.Marshal(prog)
		callback.Callback(pm, &FilePartTracker{t}, err)
	}
	p, err := time.ParseDuration(period)
	if err != nil {
		return err
	}
	tid := ftCrypto.UnmarshalTransferID(tidBytes)
	return f.ft.RegisterReceivedProgressCallback(&tid, cb, p)
}

////////////////////////////////////////////////////////////////////////////////
// Utility Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

// MaxFileNameLen returns the max number of bytes allowed for a file name.
func (f *FileTransfer) MaxFileNameLen() int {
	return f.ft.MaxFileNameLen()
}

// MaxFileTypeLen returns the max number of bytes allowed for a file type.
func (f *FileTransfer) MaxFileTypeLen() int {
	return f.ft.MaxFileTypeLen()
}

// MaxFileSize returns the max number of bytes allowed for a file.
func (f *FileTransfer) MaxFileSize() int {
	return f.ft.MaxFileSize()
}

// MaxPreviewSize returns the max number of bytes allowed for a file preview.
func (f *FileTransfer) MaxPreviewSize() int {
	return f.ft.MaxPreviewSize()
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
//  - 0 < Part does not exist
//  - 0 = unsent
//  - 1 = arrived (sender has sent a part, and it has arrived)
//  - 2 = received (receiver has received a part)
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
//  {
//   "Priority":1,
//   "Category":"Test Events",
//   "EventType":"Ping",
//   "Details":"This is an example of an event report"
//  }
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
//  - payload - JSON marshalled EventReport object
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
