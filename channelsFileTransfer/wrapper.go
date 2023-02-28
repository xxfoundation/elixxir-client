////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package channelsFileTransfer

import (
	"crypto/ed25519"
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"

	"gitlab.com/elixxir/client/v4/channels"
	"gitlab.com/elixxir/client/v4/channelsFileTransfer/store"
	"gitlab.com/elixxir/client/v4/channelsFileTransfer/store/fileMessage"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/client/v4/stoppable"
	"gitlab.com/elixxir/client/v4/xxdk"
	cryptoChannel "gitlab.com/elixxir/crypto/channel"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/elixxir/crypto/message"
	cryptoMessage "gitlab.com/elixxir/crypto/message"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/netTime"
)

// Wrapper facilitates the sending and receiving file over channels using the
// event model. It adheres to the FileTransfer interface.
type Wrapper struct {
	m  *manager
	ch channels.Manager
	ev EventModel
	me cryptoChannel.PrivateIdentity
}

// TODO: mark file as errored (or delete it) in event model on fatal error

// NewWrapper generated a new file transfer wrapper for the channel manager and
// event model. It allows for sending and receiving of files over channels.
func NewWrapper(
	user FtE2e, params Params) (*Wrapper, channels.ExtensionBuilder, error) {

	var w Wrapper

	// Create new file manager and get list of in-progress sends and receives
	fm, inProgressSends, _, err := newManager(user, params)
	if err != nil {
		return nil, nil, err
	}
	w.m = fm

	// Calculate the size of each file part
	partSize := fileMessage.
		NewPartMessage(user.GetCmix().GetMaxMessageLength()).GetPartSize()

	eb := func(e channels.EventModel, m channels.Manager,
		me cryptoChannel.PrivateIdentity) (
		[]channels.ExtensionMessageHandler, error) {
		ev, success := e.(EventModel)
		if !success {
			return nil, errors.Errorf("%T does not contain %T", e, ev)
		}

		w.ch = m
		w.ev = ev
		w.me = me

		var sentToRemove []ftCrypto.ID
		sentFileParts := make(map[ftCrypto.ID][][]byte, len(inProgressSends))

		// Lookup file data each in-progress sent file
		for _, fid := range inProgressSends {
			file, err2 := ev.GetFile(fid)
			if err2 != nil {
				jww.ERROR.Printf("[FT] Failed to get in-progress file upload "+
					"%s from event model; dropping upload: %+v", fid, err)
				sentToRemove = append(sentToRemove, fid)
			} else {
				sentFileParts[fid] = partitionFile(file.FileData, partSize)
			}
		}

		// Load file data for each in-progress file back into the file manager
		err = fm.LoadInProgressTransfers(sentFileParts, sentToRemove)
		if err != nil {
			return nil, err
		}

		return []channels.ExtensionMessageHandler{&w}, nil
	}

	return &w, eb, nil
}

// StartProcesses starts the sending threads. Adheres to the xxdk.Service type.
func (w *Wrapper) StartProcesses() (stoppable.Stoppable, error) {
	return w.m.StartProcesses()
}

// MaxFileNameLen returns the max number of bytes allowed for a file name.
func (w *Wrapper) MaxFileNameLen() int {
	return w.m.maxFileNameLen()
}

// MaxFileTypeLen returns the max number of bytes allowed for a file type.
func (w *Wrapper) MaxFileTypeLen() int {
	return w.m.maxFileTypeLen()
}

// MaxFileSize returns the max number of bytes allowed for a file.
func (w *Wrapper) MaxFileSize() int {
	return w.m.maxFileSize()
}

// MaxPreviewSize returns the max number of bytes allowed for a file preview.
func (w *Wrapper) MaxPreviewSize() int {
	return w.m.maxPreviewSize()
}

/* === Sending ============================================================== */

// Upload starts uploading the file to a new ID that can be sent to the
// specified channel when complete. To get progress information about the upload
// a SentProgressCallback must be registered.
func (w *Wrapper) Upload(fileData []byte, retry float32,
	progressCB SentProgressCallback, period time.Duration) (ftCrypto.ID, error) {

	// Return an error if the file is too large or empty
	if err := w.m.verifyFile(fileData); err != nil {
		return ftCrypto.ID{}, err
	}

	// Generate the file ID
	fid := ftCrypto.NewID(fileData)

	// Check if the file is already uploading
	st, exists := w.m.sent.GetTransfer(fid)
	if exists {
		// File upload is already in progress so the progress callback is
		// registered to the ongoing upload
		w.m.registerSentProgressCallback(st, progressCB, period)
		return fid, nil
	}

	// If the file is currently downloading, return an error
	if _, exists = w.m.received.GetTransfer(fid); exists {
		// TODO: Handle an upload that is currently downloading by adding the
		//  file data to the event model and marking the receive file transfer
		//  as complete; need to figure out how to handle file link
		return ftCrypto.ID{}, errors.Errorf("file currently downloading; " +
			"wait for process to finish to continue")
	}

	// Check if the file exists in the event model
	file, err := w.ev.GetFile(fid)
	if err != nil {
		if !channels.CheckNoMessageErr(err) {
			// Return the error
			return ftCrypto.ID{}, err
		}

		// If the file does not exist, add it to the event model and upload it
		w.ev.ReceiveFile(fid, nil, fileData, netTime.Now(), Uploading)
	} else {
		var fl FileLink
		if err = json.Unmarshal(file.FileLink, &fl); err != nil {
			return ftCrypto.ID{}, err
		}

		if fl.Expired() {
			// If the file exists and is new enough to be downloaded, call the
			// progress callback to indicate it is complete
			go progressCB(true, 0, fl.NumParts, fl.NumParts, &fl, nil, nil)
			return fid, nil
		}
	}

	// If it does not exist in storage or the event model or the file is too
	// old, then the file needs to be uploaded
	err = w.m.Send(fid, fileData, retry, w.uploadCompleteCB, progressCB, period)
	if err != nil {
		return ftCrypto.ID{}, err
	}

	return fid, nil
}

// uploadCompleteCB is called when a file upload completes. It closes out the
// file send and updates the event model.
func (w *Wrapper) uploadCompleteCB(fl FileLink) {
	fileLink, err := json.Marshal(fl)
	if err != nil {
		jww.ERROR.Printf("[FT] Failed to JSON marshal %t for file %s: %+v",
			fl, fl.FileID, err)
		return
	}

	timeNow := netTime.Now()
	status := Complete
	err = w.ev.UpdateFile(fl.FileID, &fileLink, nil, &timeNow, &status)
	if err != nil {
		jww.ERROR.Printf("[FT] Failed to update file %s: %+v", fl.FileID, err)
		return
	}
}

// Send sends the specified file info to the channel.
func (w *Wrapper) Send(channelID *id.ID, fileLink []byte, fileName,
	fileType string, preview []byte, validUntil time.Duration,
	params xxdk.CMIXParams) (message.ID, rounds.Round, ephemeral.Id, error) {

	if err := w.m.verifyFileInfo(fileName, fileType, preview); err != nil {
		return message.ID{}, rounds.Round{}, ephemeral.Id{}, err
	}

	var fl FileLink
	if err := json.Unmarshal(fileLink, &fl); err != nil {
		return message.ID{}, rounds.Round{}, ephemeral.Id{}, err
	}

	// TODO: check if a newer link exists in the event model
	if fl.Expired() {
		return message.ID{}, rounds.Round{}, ephemeral.Id{}, errors.Errorf(
			"file link expired; send occured %d ago",
			netTime.Since(fl.SentTimestamp))
	}

	fi := FileInfo{
		FileName: fileName,
		FileType: fileType,
		Preview:  preview,
		FileLink: fl,
	}

	fileInfo, err := json.Marshal(fi)
	if err != nil {
		return message.ID{}, rounds.Round{}, ephemeral.Id{}, err
	}

	return w.ch.SendGeneric(channelID, channels.FileTransfer,
		fileInfo, validUntil, true, params.CMIX)
}

// RegisterSentProgressCallback registers the callback to the given file
// described in the marshalled FileInfo.
func (w *Wrapper) RegisterSentProgressCallback(fileID ftCrypto.ID,
	progressCB SentProgressCallback, period time.Duration) error {
	st, exists := w.m.sent.GetTransfer(fileID)
	if exists {
		w.m.registerSentProgressCallback(st, progressCB, period)
		return nil
	}

	file, err := w.ev.GetFile(fileID)
	if err != nil {
		if !channels.CheckNoMessageErr(err) {
			return err
		}

		return errors.Errorf(errNoSentTransfer, fileID)
	}

	var fl FileLink
	if err = json.Unmarshal(file.FileLink, &fl); err != nil {
		return err
	}

	switch file.Status {
	case Complete:
		// TODO fix missing fields
		progressCB(true, 0, fl.NumParts, fl.NumParts, &fl, nil, nil)
	}

	return w.m.RegisterSentProgressCallback(fileID, progressCB, period)
}

// RetrySend retries uploading a failed file upload. Returns an error if the
// transfer has not run out of retries.
//
// This function should be called once a transfer errors out (as reported by
// the progress callback).
// TODO: write function
func (w *Wrapper) RetrySend(fileID ftCrypto.ID) error {

	return nil
}

// CloseSend deletes a file from the internal storage once a transfer has
// completed or reached the retry limit. If neither of those condition are
// met, an error is returned.
//
// This function should be called once a transfer completes or errors out
// (as reported by the progress callback).
// TODO: handle deletion from event model
func (w *Wrapper) CloseSend(fileID ftCrypto.ID) error {

	st, exists := w.m.sent.GetTransfer(fileID)
	if exists {
		if err := w.m.CloseSend(fileID); err != nil {
			return err
		}
	} else {
		w.ev.GetFile(fileID)
	}

	w.ev.DeleteFile(fileID)

	return w.m.CloseSend(fileID)
}

/* === Receiving ============================================================ */

// Download beings the download of the file described in the marshalled
// FileInfo. The progress of the download is reported on the progress callback.
// TODO: what happens when downloading the same file again?
func (w *Wrapper) Download(fileInfo []byte, progressCB ReceivedProgressCallback,
	period time.Duration) (ftCrypto.ID, error) {

	var fi FileInfo
	if err := json.Unmarshal(fileInfo, &fi); err != nil {
		return ftCrypto.ID{}, err
	}

	// Check if the file is already downloading
	rt, exists := w.m.received.GetTransfer(fi.FileID)
	if exists {
		// File download is already in progress so the progress callback is
		// registered to the ongoing download
		w.m.registerReceivedProgressCallback(rt, progressCB, period)
		return fi.FileID, nil
	}

	// If the file is currently uploading, return an error
	if _, exists = w.m.sent.GetTransfer(fi.FileID); exists {
		// TODO: Handle a downloading that is currently uploading by immediately
		//  marking it as complete; need to figure out how to handle file link
		return ftCrypto.ID{}, errors.Errorf("file currently downloading; " +
			"wait for process to finish to continue")
	}

	// Check if the file exists in the event model
	file, err := w.ev.GetFile(fi.FileID)
	if err != nil {
		if !channels.CheckNoMessageErr(err) {
			// Return the error
			return ftCrypto.ID{}, err
		}

		if fi.Expired() {
			return ftCrypto.ID{}, errors.Errorf(
				"file link expired; send occured %d ago",
				netTime.Since(fi.SentTimestamp))
		}

		fileLink, err2 := json.Marshal(fi.FileLink)
		if err2 != nil {
			return ftCrypto.ID{}, err2
		}

		// If the file does not exist, add it to the event model and download it
		w.ev.ReceiveFile(fi.FileID, fileLink, nil, netTime.Now(), Uploading)

		// Start downloading file
		rt, err = w.m.handleIncomingTransfer(&fi, progressCB, period)
		if err != nil {
			return ftCrypto.ID{}, err
		}

		// Register callback to update event model once download is complete
		w.m.registerReceivedProgressCallback(rt, w.downloadCompleteCB(rt), 0)
	}

	// TODO: if the file exists but is not marked complete, restart the download
	if file.Status != Complete {
		return ftCrypto.ID{}, errors.Errorf("file not marked as complete")
	}

	// If the file exists, call the progress callback to indicate it is
	// complete
	go progressCB(true, fi.NumParts, fi.NumParts, &fi, nil, nil)
	return fi.FileID, nil
}

// downloadCompleteCB is called when a file download completes. It receives the
// full file (removing it from the file manager) and updates the event model.
func (w *Wrapper) downloadCompleteCB(
	rt *store.ReceivedTransfer) ReceivedProgressCallback {
	return func(completed bool, _, _ uint16, _ ReceivedTransfer,
		_ FilePartTracker, err error) {
		if err != nil {
			jww.ERROR.Printf("[FT] Error downloading file %s (%q): %+v",
				rt.GetFileID(), rt.FileName(), err)
			return
		}

		if completed {
			// Get completed file
			fileData, err2 := w.m.receive(rt)
			if err2 != nil {
				jww.ERROR.Printf("[FT] Failed to get complete file data for "+
					"%s (%q): %+v", rt.GetFileID(), rt.FileName(), err2)
				return
			}

			// Store completed file in event model
			now := netTime.Now()
			status := Complete
			err = w.ev.UpdateFile(rt.GetFileID(), nil, &fileData, &now, &status)
			if err != nil {
				jww.ERROR.Printf("[FT] Failed to update downloaded file %s "+
					"(%q) in event model: %+v", rt.GetFileID(), rt.FileName(), err)
			}
		}
	}
}

// RegisterReceivedProgressCallback registers the callback to the given file ID.
func (w *Wrapper) RegisterReceivedProgressCallback(fileID ftCrypto.ID,
	progressCB ReceivedProgressCallback, period time.Duration) error {
	return w.m.RegisterReceivedProgressCallback(fileID, progressCB, period)
}

////////////////////////////////////////////////////////////////////////////////
// ExtensionMessageHandler                                                    //
////////////////////////////////////////////////////////////////////////////////

// The functions below adhere to the channels.ExtensionMessageHandler interface.

// GetType returns the channels.FileTransfer message type.
func (w *Wrapper) GetType() channels.MessageType {
	return channels.FileTransfer
}

// GetProperties returns debugging and pre-filtering information.
func (w *Wrapper) GetProperties() (
	name string, userSpace, adminSpace, mutedSpace bool) {
	return "fileTransfer", true, true, false
}

// Handle handles the file transfer file info message.
func (w *Wrapper) Handle(channelID *id.ID, messageID cryptoMessage.ID,
	messageType channels.MessageType, nickname string, content, _ []byte,
	pubKey ed25519.PublicKey, dmToken uint32, codeset uint8, timestamp,
	_ time.Time, lease time.Duration, _ id.Round, round rounds.Round,
	_ channels.SentStatus, _, hidden bool) uint64 {

	fi, err := UnmarshalFileInfo(content)
	if err != nil {
		jww.ERROR.Printf("[CH] Failed to text unmarshal file information %s "+
			"from %x on channel %s, type %s, ts: %s, lease: %s, round: %d: %+v",
			messageID, pubKey, channelID, messageType, timestamp, lease,
			round.ID, err)
		return 0
	}

	jww.INFO.Printf("[CH] Received file transfer %s from %x on %s",
		fi.FileID, pubKey, channelID)

	return w.ev.ReceiveMessage(channelID, messageID, nickname, string(content),
		pubKey, dmToken, codeset, timestamp, lease, round, messageType,
		channels.Delivered, hidden)
}
