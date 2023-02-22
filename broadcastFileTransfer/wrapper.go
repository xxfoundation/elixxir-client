////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package broadcastFileTransfer

import (
	"crypto/ed25519"
	"time"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"

	"gitlab.com/elixxir/client/v4/broadcastFileTransfer/store"
	"gitlab.com/elixxir/client/v4/broadcastFileTransfer/store/fileMessage"
	"gitlab.com/elixxir/client/v4/channels"
	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/client/v4/stoppable"
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
func NewWrapper(user FtE2e, params Params) (
	*Wrapper, channels.ExtensionBuilder, error) {

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

		// Lookup file data each in-progress sent file
		var sentToRemove []ftCrypto.ID
		sentFileParts := make(map[ftCrypto.ID][][]byte, len(inProgressSends))
		for _, fid := range inProgressSends {
			_, fileData, err2 := ev.GetFile(fid)
			if err2 != nil {
				jww.ERROR.Printf("[FT] Failed to get in-progress file upload "+
					"%s from event model; dropping upload: %+v", fid, err)
				sentToRemove = append(sentToRemove, fid)
			} else {
				sentFileParts[fid] = partitionFile(fileData, partSize)
			}
		}

		// Load file data for each in-progress file back into the file manager
		err = fm.LoadInProgressTransfers(
			sentFileParts, sentToRemove)
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
// a SentProgressCallback but be registered.
func (w *Wrapper) Upload(channelID *id.ID, fileName, fileType string,
	fileData []byte, retry float32, preview []byte,
	progressCB SentProgressCallback, period time.Duration) (ftCrypto.ID, error) {

	// Initiate file send
	fid, err := w.m.Send(fileName, fileType, fileData, retry, preview,
		w.uploadCompleteCB(channelID), progressCB, period)
	if err != nil {
		return ftCrypto.ID{}, err
	}

	// Store file in event model
	nickname, _ := w.ch.GetNickname(channelID)
	w.ev.ReceiveFileMessage(channelID, fid, nickname, nil, fileData, w.me.PubKey,
		w.me.GetDMToken(), 0, netTime.Now(), 0, rounds.Round{},
		channels.FileTransfer, channels.SendProcessing, false)

	return fid, nil
}

// uploadCompleteCB is called when a file upload completes. It closes out the
// file send and updates the event model.
func (w *Wrapper) uploadCompleteCB(channelID *id.ID) SendCompleteCallback {
	return func(fi FileInfo) {
		fileInfo, err := fi.Marshal()
		if err != nil {
			jww.ERROR.Printf("[FT] Failed to marshal FileInfo for file %s "+
				"(%q) on channel %s: %+v", fi.FID, fi.FileName, channelID, err)
			return
		}

		timeNow := netTime.Now()
		status := channels.SendProcessingComplete
		err = w.ev.UpdateFile(
			fi.FID, &fileInfo, nil, &timeNow, nil, nil, nil, &status)
		if err != nil {
			jww.ERROR.Printf("[FT] Failed to update file %s (%q) on channel "+
				"%s: %+v", fi.FID, fi.FileName, channelID, err)
			return
		}
	}
}

// Send sends the specified file info to the channel.
func (w *Wrapper) Send(channelID *id.ID, fileInfo []byte,
	validUntil time.Duration, params cmix.CMIXParams) (
	message.ID, rounds.Round, ephemeral.Id, error) {

	return w.ch.SendGeneric(channelID, channels.FileTransfer, fileInfo,
		validUntil, true, params)
}

// RegisterSentProgressCallback registers the callback to the given file
// described in the marshalled FileInfo.
func (w *Wrapper) RegisterSentProgressCallback(fileID ftCrypto.ID,
	progressCB SentProgressCallback, period time.Duration) error {

	return w.m.RegisterSentProgressCallback(fileID, progressCB, period)
}

/* === Receiving ============================================================ */

// Download beings the download of the file described in the marshalled
// FileInfo. The progress of the download is reported on the progress callback.
func (w *Wrapper) Download(fileInfo []byte, progressCB ReceivedProgressCallback,
	period time.Duration) (ftCrypto.ID, error) {

	// Add file to file manager
	rt, fi, err := w.m.handleIncomingTransfer(fileInfo, progressCB, period)
	if err != nil {
		return ftCrypto.ID{}, err
	}

	// Register callback to update event model once download is complete
	w.m.registerReceivedProgressCallback(rt, w.downloadCompleteCB(rt), 0)

	status := channels.ReceptionProcessing
	return fi.FID, w.ev.UpdateFile(fi.FID, nil, nil, nil, nil, nil, nil, &status)
}

// downloadCompleteCB is called when a file download completes. It receives the
// full file (removing it from the file manager) and updates the event model.
func (w *Wrapper) downloadCompleteCB(
	rt *store.ReceivedTransfer) ReceivedProgressCallback {
	return func(completed bool, _, _ uint16, _ ReceivedTransfer,
		_ FilePartTracker, err error) {
		if err != nil {
			jww.ERROR.Printf("[FT] Error downloading file %s (%q): %+v",
				rt.FileID(), rt.FileName(), err)
			return
		}

		if completed {
			// Get completed file
			fileData, err2 := w.m.receive(rt)
			if err2 != nil {
				jww.ERROR.Printf("[FT] Failed to get complete file data for "+
					"%s (%q): %+v", rt.FileID(), rt.FileName(), err2)
				return
			}

			// Store completed file in event model
			now := netTime.Now()
			status := channels.ReceptionProcessingComplete
			err = w.ev.UpdateFile(
				rt.FileID(), nil, &fileData, &now, nil, nil, nil, &status)
			if err != nil {
				jww.ERROR.Printf("[FT] Failed to update downloaded file %s "+
					"(%q) in event model: %+v", rt.FileID(), rt.FileName(), err)
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

	ti, err := UnmarshalFileInfo(content)
	if err != nil {
		jww.ERROR.Printf("[CH] Failed to text unmarshal file information %s "+
			"from %x on channel %s, type %s, ts: %s, lease: %s, round: %d: %+v",
			messageID, pubKey, channelID, messageType, timestamp, lease,
			round.ID, err)
		return 0
	}

	jww.INFO.Printf("[CH] Received file transfer %s from %x on %s",
		ti.FID, pubKey, channelID)

	return w.ev.ReceiveFileMessage(channelID, ti.FID, nickname, content, nil,
		pubKey, dmToken, codeset, timestamp, lease, round, messageType,
		channels.Delivered, hidden)
}
