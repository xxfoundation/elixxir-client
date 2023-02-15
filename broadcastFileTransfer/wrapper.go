////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package broadcastFileTransfer

import (
	"crypto/ed25519"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/broadcastFileTransfer/store/fileMessage"
	"gitlab.com/elixxir/client/v4/channels"
	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	cryptoChannel "gitlab.com/elixxir/crypto/channel"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/elixxir/crypto/message"
	cryptoMessage "gitlab.com/elixxir/crypto/message"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/netTime"
	"time"
)

type Wrapper struct {
	m  *manager
	ch channels.Manager
	ev EventModel
	me cryptoChannel.PrivateIdentity
}

// NewWrapper generated a new file transfer wrapper for the channel manager and
// event model. It allows for sending and receiving of files over channels.
func NewWrapper(user FtE2e, params Params) (
	*Wrapper, channels.ExtensionBuilder, error) {

	var w *Wrapper

	// Create new file manager and get list of in-progress sends and receives
	fm, inProgressSends, inProgressReceives, err := NewManager(user, params)
	if err != nil {
		return nil, nil, err
	}
	w.m = fm.(*manager)

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
				jww.ERROR.Printf("[FT] Failed to get in-progress upload %s "+
					"from event model; dropping upload: %+v", fid, err)
				sentToRemove = append(sentToRemove, fid)
			} else {
				sentFileParts[fid] = partitionFile(fileData, partSize)
			}
		}

		// Lookup file data each in-progress received file
		var receivedToRemove []ftCrypto.ID
		receivedFileParts := make(map[ftCrypto.ID][][]byte, len(inProgressReceives))
		for _, fid := range inProgressReceives {
			_, fileData, err2 := ev.GetFile(fid)
			if err2 != nil {
				jww.ERROR.Printf("[FT] Failed to get in-progress download %s "+
					"from event model; dropping download: %+v", fid, err)
				receivedToRemove = append(receivedToRemove, fid)
			} else {
				sentFileParts[fid] = partitionFile(fileData, partSize)
			}
		}

		// Load file data for each in-progress file back into the file manager
		err = fm.LoadInProgressTransfers(
			sentFileParts, receivedFileParts, sentToRemove, receivedToRemove)
		if err != nil {
			return nil, err
		}

		return []channels.ExtensionMessageHandler{w}, nil
	}

	return w, eb, nil
}

func (w *Wrapper) Upload(channelID *id.ID, fileName, fileType string,
	fileData []byte, retry float32, preview []byte,
	progressCB SentProgressCallback, period time.Duration,
	validUntil time.Duration) (uint64, error) {

	var fid ftCrypto.ID
	completeCB := func(fi FileInfo) {
		defer func() {
			if err := w.m.CloseSend(fi.FID); err != nil {
				jww.ERROR.Printf("[FT] Failed to close send of file %s on "+
					"channel %s: %+v", fid, channelID, err)
			}
		}()

		fileInfo, err := fi.Marshal()
		if err != nil {
			jww.ERROR.Printf("[FT] Failed to marshal file info for file "+
				"%s on channel %s: %+v", fid, channelID, err)
			return
		}

		timeNow := netTime.Now()
		err = w.ev.UpdateFile(fid, &fileInfo, nil, &timeNow, nil, nil, nil, nil)
		if err != nil {
			jww.ERROR.Printf("[FT] Failed to update file %s on channel %s: %+v",
				fid, channelID, err)
			return
		}
	}

	var err error
	fid, err = w.m.Send(fileName, fileType, fileData, retry, preview,
		completeCB, progressCB, period)
	if err != nil {
		return 0, err
	}

	pubKey := w.ch.GetIdentity().PubKey
	nickname, _ := w.ch.GetNickname(channelID)
	dmToken := w.me.GetDMToken()
	uuid := w.ev.ReceiveFileMessage(channelID, fid, nickname, nil, fileData,
		pubKey, dmToken, 0, netTime.Now(), validUntil, rounds.Round{},
		channels.FileTransfer, channels.SendProcessing, false)

	return uuid, nil
}

func (w *Wrapper) Send(channelID *id.ID, fileInfo []byte,
	validUntil time.Duration, params cmix.CMIXParams) (
	message.ID, rounds.Round, ephemeral.Id, error) {

	return w.ch.SendGeneric(channelID, channels.FileTransfer, fileInfo,
		validUntil, true, params)
}

func (w *Wrapper) Download(fileInfo []byte,
	progressCB ReceivedProgressCallback, period time.Duration) error {

	fid, _, err := w.m.HandleIncomingTransfer(fileInfo, progressCB, period)
	if err != nil {
		return err
	}

	progressCB2 := func(fileData []byte, err error) {
		if err != nil {
			jww.ERROR.Printf("[FT] Failed to update download progress for "+
				"file %s: %+v", fid, err)
		} else {
			err = w.ev.UpdateFile(fid, nil, &fileData, nil, nil, nil, nil, nil)
			if err != nil {
				jww.ERROR.Printf("[FT] Failed to update download progress for "+
					"file %s: %+v", fid, err)
			}
		}
	}

	err = w.m.registerReceivedPartsCallback(fid, progressCB2, 0)
	if err != nil {
		return err
	}

	status := channels.ReceptionProcessing
	return w.ev.UpdateFile(fid, nil, nil, nil, nil, nil, nil, &status)
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
