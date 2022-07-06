package bindings

import (
	"encoding/json"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/fileTransfer"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

/* File Transfer Structs and Interfaces */

// FileTransfer object is a bindings-layer struct which wraps a fileTransfer.FileTransfer interface
type FileTransfer struct {
	ft    fileTransfer.FileTransfer
	e2eCl *E2e
}

// ReceivedFile is a public struct which represents the contents of an incoming file
// Example JSON:
// {
//  "TransferID":"B4Z9cwU18beRoGbk5xBjbcd5Ryi9ZUFA2UBvi8FOHWo=", // ID of the incoming transfer for receiving
//  "SenderID":"emV6aW1hAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD",   // ID of sender of incoming file
//  "Preview":"aXQncyBtZSBhIHByZXZpZXc=",                        // Preview of the incoming file
//  "Name":"testfile.txt",                                       // Name of incoming file
//  "Type":"text file",                                          // Incoming file type
//  "Size":2048                                                  // Incoming file size
// }
type ReceivedFile struct {
	TransferID []byte
	SenderID   []byte
	Preview    []byte
	Name       string
	Type       string
	Size       int
}

// FileSend is a public struct which represents a file to be transferred
// {
//  "Name":"testfile.txt",  														// File name
//  "Type":"text file",     														// File type
//  "Preview":"aXQncyBtZSBhIHByZXZpZXc=",  											// Preview of contents
//  "Contents":"VGhpcyBpcyB0aGUgZnVsbCBjb250ZW50cyBvZiB0aGUgZmlsZSBpbiBieXRlcw==" 	// Full contents of the file
// }
type FileSend struct {
	Name     string
	Type     string
	Preview  []byte
	Contents []byte
}

// Progress is a public struct which represents the progress of an in-progress file transfer
// Example JSON:
// {"Completed":false,	// Status of transfer (true if done)
//  "Transmitted":128,	// Bytes transferred so far
//  "Total":2048,		// Total size of file
//  "Err":null			// Error status (if any)
// }
type Progress struct {
	Completed   bool
	Transmitted int
	Total       int
	Err         error
}

// ReceiveFileCallback is a bindings-layer interface which is called when a file is received
// Accepts the result of calling json.Marshal on a ReceivedFile struct
type ReceiveFileCallback interface {
	Callback(payload []byte, err error)
}

// FileTransferSentProgressCallback is a bindings-layer interface which is called with the progress of a sending file
// Accepts the result of calling json.Marshal on a Progress struct & a FilePartTracker interface
type FileTransferSentProgressCallback interface {
	Callback(payload []byte, t *FilePartTracker, err error)
}

// FileTransferReceiveProgressCallback is a bindings-layer interface which is called with the progress of a received file
// Accepts the result of calling json.Marshal on a Progress struct & a FilePartTracker interface
type FileTransferReceiveProgressCallback interface {
	Callback(payload []byte, t *FilePartTracker, err error)
}

/* Main functions */

// InitFileTransfer creates a bindings-level File Transfer manager
// Accepts client ID, ReceiveFileCallback and a ReporterFunc
func InitFileTransfer(e2eID int) (*FileTransfer, error) {
	paramsJSON := GetDefaultFileTransferParams()

	// Get bindings client from singleton
	e2eCl, err := e2eTrackerSingleton.get(e2eID)
	if err != nil {
		return nil, err
	}

	// Client info
	myID := e2eCl.api.GetReceptionIdentity().ID
	rng := e2eCl.api.GetRng()

	params, err := parseFileTransferParams(paramsJSON)
	if err != nil {
		return nil, err
	}

	// Create file transfer manager
	m, err := fileTransfer.NewManager(params, myID,
		e2eCl.api.GetCmix(), e2eCl.api.GetStorage(), rng)

	// Add file transfer processes to client services tracking
	err = e2eCl.api.AddService(m.StartProcesses)
	if err != nil {
		return nil, err
	}

	// Return wrapped manager
	return &FileTransfer{ft: m, e2eCl: e2eCl}, nil
}

// Send is the bindings-level function for sending a File
func (f *FileTransfer) Send(payload, recipientID []byte, retry float32,
	period string, callback FileTransferSentProgressCallback) ([]byte, error) {
	paramsJSON := GetDefaultE2EParams()
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

// Receive returns the full file on the completion of the transfer.
// It deletes internal references to the data and unregisters any attached
// progress callback. Returns an error if the transfer is not complete, the
// full file cannot be verified, or if the transfer cannot be found.
//
// Receive can only be called once the progress callback returns that the
// file transfer is complete.
func (f *FileTransfer) Receive(tidBytes []byte) ([]byte, error) {
	tid := ftCrypto.UnmarshalTransferID(tidBytes)
	return f.ft.Receive(&tid)
}

// CloseSend deletes a file from the internal storage once a transfer has
// completed or reached the retry limit. Returns an error if the transfer
// has not run out of retries.
//
// This function should be called once a transfer completes or errors out
// (as reported by the progress callback).
func (f *FileTransfer) CloseSend(tidBytes []byte) error {
	tid := ftCrypto.UnmarshalTransferID(tidBytes)
	return f.ft.CloseSend(&tid)
}

/* Callback registration functions */

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

func (f *FileTransfer) RegisterReceivedProgressCallback(tidBytes []byte, callback FileTransferReceiveProgressCallback, period string) error {
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

/* Utility Functions */

func (f *FileTransfer) MaxFileNameLen() int {
	return f.ft.MaxFileNameLen()
}

func (f *FileTransfer) MaxFileTypeLen() int {
	return f.ft.MaxFileTypeLen()
}

func (f *FileTransfer) MaxFileSize() int {
	return f.ft.MaxFileSize()
}

func (f *FileTransfer) MaxPreviewSize() int {
	return f.ft.MaxPreviewSize()
}

////////////////////////////////////////////////////////////////////////////////
// File Part Tracker                                                          //
////////////////////////////////////////////////////////////////////////////////

// FilePartTracker contains the interfaces.FilePartTracker.
type FilePartTracker struct {
	m fileTransfer.FilePartTracker
}

// GetPartStatus returns the status of the file part with the given part number.
// The possible values for the status are:
// 0 = unsent
// 1 = sent (sender has sent a part, but it has not arrived)
// 2 = arrived (sender has sent a part, and it has arrived)
// 3 = received (receiver has received a part)
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

// EventReport is a public struct which represents the contents of an event report
// Example JSON:
// {"Priority":1,
//  "Category":"Test Events",
//  "EventType":"Ping",
//  "Details":"This is an example of an event report"
// }
type EventReport struct {
	Priority  int
	Category  string
	EventType string
	Details   string
}

// ReporterFunc is a bindings-layer interface which receives info from the Event Manager
// Accepts result of json.Marshal on an EventReport object
type ReporterFunc interface {
	Report(payload []byte, err error)
}

// reporter is the internal struct to match the event.Reporter interface
type reporter struct {
	r ReporterFunc
}

// Report matches the event.Reporter interface, wraps the info in an EventReport struct
// and passes the marshalled struct to the internal callback
func (r *reporter) Report(priority int, category, evtType, details string) {
	rep := &EventReport{
		Priority:  priority,
		Category:  category,
		EventType: evtType,
		Details:   details,
	}
	r.r.Report(json.Marshal(rep))
}
