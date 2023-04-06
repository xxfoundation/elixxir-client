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
	channelsFT "gitlab.com/elixxir/client/v4/channelsFileTransfer"
	ftCrypto "gitlab.com/elixxir/crypto/fileTransfer"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

// ChannelsFileTransfer manages the sending and receiving of file via channels.
// Refer to [channelsFileTransfer.FileTransfer] for additional documentation.
type ChannelsFileTransfer struct {
	api channelsFT.FileTransfer
}

// InitChannelsFileTransfer creates a file transfer manager for channels.
//
// Parameters:
//   - e2eID - ID of [E2e] object in tracker.
//   - paramsJson - JSON of [channelsFileTransfer.Params].
//
// Returns:
//   - New [ChannelsFileTransfer] object.
func InitChannelsFileTransfer(
	e2eID int, paramsJson []byte) (*ChannelsFileTransfer, error) {
	jww.INFO.Printf("[FT] InitChannelsFileTransfer(e2eID:%d params:%s)",
		e2eID, paramsJson)

	user, err := e2eTrackerSingleton.get(e2eID)
	if err != nil {
		return nil, err
	}

	var params channelsFT.Params
	if err = json.Unmarshal(paramsJson, &params); err != nil {
		return nil, errors.Wrapf(err, "could not unmarshal %T", params)
	}

	// Create file transfer manager
	w, _, err := channelsFT.NewWrapper(user.api, params)
	if err != nil {
		return nil, errors.Errorf(
			"could not create new file transfer manager: %+v", err)
	}

	// Add file transfer processes to services tracking
	err = user.api.AddService(w.StartProcesses)
	if err != nil {
		return nil, errors.Wrap(err, "could not start file transfer service")
	}

	// Return wrapped manager
	return &ChannelsFileTransfer{w}, nil
}

// MaxFileNameLen returns the max number of bytes allowed for a file name.
func (ft *ChannelsFileTransfer) MaxFileNameLen() int {
	return ft.api.MaxFileNameLen()
}

// MaxFileTypeLen returns the max number of bytes allowed for a file type.
func (ft *ChannelsFileTransfer) MaxFileTypeLen() int {
	return ft.api.MaxFileTypeLen()
}

// MaxFileSize returns the max number of bytes allowed for a file.
func (ft *ChannelsFileTransfer) MaxFileSize() int {
	return ft.api.MaxFileSize()
}

// MaxPreviewSize returns the max number of bytes allowed for a file preview.
func (ft *ChannelsFileTransfer) MaxPreviewSize() int {
	return ft.api.MaxPreviewSize()
}

////////////////////////////////////////////////////////////////////////////////
// Uploading/Sending                                                          //
////////////////////////////////////////////////////////////////////////////////

// Upload starts uploading the file to a new ID that can be sent to the
// specified channel when complete. To get progress information about the
// upload, a [FtSentProgressCallback] must be registered. All errors returned on
// the callback are fatal and the user must take action to either
// [ChannelsFileTransfer.RetryUpload] or [ChannelsFileTransfer.CloseSend].
//
// The file is added to the event model at the returned file ID with the status
// [channelsFileTransfer.Uploading]. Once the upload is complete, the file link
// is added to the event model with the status [channelsFileTransfer.Complete].
//
// The [FtSentProgressCallback] only indicates the progress of the file upload,
// not the status of the file in the event model. You must rely on updates from
// the event model to know when it can be retrieved.
//
// Parameters:
//   - fileData - File contents. Max size defined by
//     [ChannelsFileTransfer.MaxFileSize].
//   - retry - The number of sending retries allowed on send failure (e.g. a
//     retry of 2.0 with 6 parts means 12 total possible sends).
//   - progressCB - A callback that reports the progress of the file upload.
//     The callback is called once on initialization, on every progress
//     update (or less if restricted by the period), or on fatal error.
//   - periodMS - A progress callback will be limited from triggering only once
//     per period, in milliseconds.
//
// Returns:
//   - Marshalled bytes of [fileTransfer.ID] that uniquely identifies the file.
func (ft *ChannelsFileTransfer) Upload(fileData []byte, retry float32,
	progressCB FtSentProgressCallback, periodMS int) ([]byte, error) {
	jww.INFO.Printf("[FT] Uploading file transfer")

	cb := func(completed bool, sent, received, total uint16,
		st channelsFT.SentTransfer, t channelsFT.FilePartTracker, err error) {
		sp := &FtSentProgress{
			st.GetFileID(), completed, int(sent), int(received), int(total)}
		data, err2 := json.Marshal(sp)
		if err2 != nil {
			jww.FATAL.Panicf("[FT] Failed to JSON marshal %T: %+v", sp, err)
		}
		progressCB.Callback(data, &ChFilePartTracker{t}, err)
	}

	period := time.Duration(periodMS) * time.Millisecond

	fid, err := ft.api.Upload(fileData, retry, cb, period)
	if err != nil {
		return nil, err
	}

	return fid.Marshal(), nil
}

// Send sends the specified file info to the channel. Once a file is uploaded
// via [ChannelsFileTransfer.Upload], its file info (found in the event model)
// can be sent to any channel.
//
// Parameters:
//   - channelIdBytes - Marshalled bytes of the channel's [id.ID] to send the
//     file to.
//   - fileLinkJSON - JSON of [channelsFileTransfer.Link] stored in the event
//     model.
//   - fileName - Human-readable file name. Max length defined by
//     [MaxFileNameLen].
//   - fileType - Shorthand that identifies the type of file. Max length defined
//     by [MaxFileTypeLen].
//   - preview - A preview of the file data (e.g. a thumbnail). Max size defined
//     by [MaxPreviewSize].
//   - validUntilMS - The duration, in milliseconds, that the file is available
//     in the channel. For the maximum amount of time, use [ValidForever].
//   - cmixParamsJSON - JSON of [xxdk.CMIXParams]. If left empty,
//     [GetDefaultCMixParams] will be used internally.
//
// Returns:
//   - JSON of [ChannelSendReport].
func (ft *ChannelsFileTransfer) Send(channelIdBytes, fileLinkJSON []byte,
	fileName, fileType string, preview []byte,
	validUntilMS int, cmixParamsJSON []byte) ([]byte, error) {
	jww.INFO.Printf("[FT] Sending file transfer to channel %s.",
		base64.StdEncoding.EncodeToString(channelIdBytes))

	channelID, err := id.Unmarshal(channelIdBytes)
	if err != nil {
		return nil, err
	}

	validUntil := time.Duration(validUntilMS) * time.Millisecond

	params, err := parseCMixParams(cmixParamsJSON)
	if err != nil {
		return nil, err
	}

	msgID, round, ephID, err := ft.api.Send(
		channelID, fileLinkJSON, fileName, fileType, preview, validUntil, params)
	if err != nil {
		return nil, err
	}

	return constructChannelSendReport(&msgID, round.ID, &ephID)
}

// RegisterSentProgressCallback allows for the registration of a callback to
// track the progress of an individual file upload. A [FtSentProgressCallback]
// is auto registered on [ChannelsFileTransfer.Send]; this function should be
// called when resuming clients or registering extra callbacks.
//
// The callback will be called immediately when added to report the current
// progress of the transfer. It will then call every time a file part arrives,
// the transfer completes, or a fatal error occurs. It is called at most once
// every period regardless of the number of progress updates.
//
// In the event that the client is closed and resumed, this function must be
// used to re-register any callbacks previously registered with this function or
// [ChannelsFileTransfer.Send].
//
// The [FtSentProgressCallback] only indicates the progress of the file upload,
// not the status of the file in the event model. You must rely on updates from
// the event model to know when it can be retrieved.
//
// Parameters:
//   - fileIDBytes - Marshalled bytes of the file's [fileTransfer.ID].
//   - progressCB - A callback that reports the progress of the file upload.
//     The callback is called once on initialization, on every progress
//     update (or less if restricted by the period), or on fatal error.
//   - periodMS - A progress callback will be limited from triggering only
//     once per period, in milliseconds.
func (ft *ChannelsFileTransfer) RegisterSentProgressCallback(fileIDBytes []byte,
	progressCB FtSentProgressCallback, periodMS int) error {
	jww.INFO.Printf("[FT] Registering SentProgressCallback to %s.",
		base64.StdEncoding.EncodeToString(fileIDBytes))

	fileID, err := ftCrypto.UnmarshalID(fileIDBytes)
	if err != nil {
		return errors.Errorf("failed to unmarshal file ID: %+v", err)
	}

	cb := func(completed bool, sent, received, total uint16,
		st channelsFT.SentTransfer, t channelsFT.FilePartTracker, err error) {
		sp := &FtSentProgress{
			st.GetFileID(), completed, int(sent), int(received), int(total)}
		data, err2 := json.Marshal(sp)
		if err2 != nil {
			jww.FATAL.Panicf("[FT] Failed to JSON marshal %T: %+v", sp, err)
		}
		progressCB.Callback(data, &ChFilePartTracker{t}, err)
	}

	period := time.Duration(periodMS) * time.Millisecond

	return ft.api.RegisterSentProgressCallback(fileID, cb, period)
}

// RetryUpload retries uploading a failed file upload. Returns an error if the
// transfer has not failed.
//
// This function should be called once a transfer errors out (as reported by the
// progress callback).
//
// A new progress callback must be registered on retry. Any previously
// registered callbacks are defunct when the upload fails.
//
// Parameters:
//   - fileIDBytes - Marshalled bytes of the file's [fileTransfer.ID].
//   - progressCB - A callback that reports the progress of the file upload.
//     The callback is called once on initialization, on every progress
//     update (or less if restricted by the period), or on fatal error.
//   - periodMS - A progress callback will be limited from triggering only
//     once per period, in milliseconds.
func (ft *ChannelsFileTransfer) RetryUpload(fileIDBytes []byte,
	progressCB FtSentProgressCallback, periodMS int) error {
	jww.INFO.Printf("[FT] Retry send of %s.",
		base64.StdEncoding.EncodeToString(fileIDBytes))

	fileID, err := ftCrypto.UnmarshalID(fileIDBytes)
	if err != nil {
		return errors.Errorf("failed to unmarshal file ID: %+v", err)
	}

	cb := func(completed bool, sent, received, total uint16,
		st channelsFT.SentTransfer, t channelsFT.FilePartTracker, err error) {
		sp := &FtSentProgress{
			st.GetFileID(), completed, int(sent), int(received), int(total)}
		data, err2 := json.Marshal(sp)
		if err2 != nil {
			jww.FATAL.Panicf("[FT] Failed to JSON marshal %T: %+v", sp, err)
		}
		progressCB.Callback(data, &ChFilePartTracker{t}, err)
	}

	period := time.Duration(periodMS) * time.Millisecond

	return ft.api.RetryUpload(fileID, cb, period)
}

// CloseSend deletes a file from the internal storage once a transfer has
// completed or reached the retry limit. If neither of those condition are met,
// an error is returned.
//
// This function should be called once a transfer completes or errors out (as
// reported by the progress callback).
//
// Parameters:
//   - fileIDBytes - Marshalled bytes of the file's [fileTransfer.ID].
func (ft *ChannelsFileTransfer) CloseSend(fileIDBytes []byte) error {
	jww.INFO.Printf("[FT] Close send of %s.",
		base64.StdEncoding.EncodeToString(fileIDBytes))

	fileID, err := ftCrypto.UnmarshalID(fileIDBytes)
	if err != nil {
		return errors.Errorf("failed to unmarshal file ID: %+v", err)
	}

	return ft.api.CloseSend(fileID)
}

////////////////////////////////////////////////////////////////////////////////
// Downloading                                                                //
////////////////////////////////////////////////////////////////////////////////

// Download begins the download of the file described in the marshalled
// [channelsFileTransfer.FileInfo]. The progress of the download is reported on
// the [FtReceivedProgressCallback].
//
// Once the download completes, the file will be stored in the event model with
// the given file ID and with the status [channels.ReceptionProcessingComplete].
//
// The [FtReceivedProgressCallback] only indicates the progress of the file
// download, not the status of the file in the event model. You must rely on
// updates from the event model to know when it can be retrieved.
//
// Parameters:
//   - fileInfoJSON - The JSON of [channelsFileTransfer.FileInfo] received on a
//     channel.
//   - progressCB - A callback that reports the progress of the file download.
//     The callback is called once on initialization, on every progress update
//     (or less if restricted by the period), or on fatal error.
//   - periodMS - A progress callback will be limited from triggering only once
//     per period, in milliseconds.
//
// Returns:
//   - Marshalled bytes of [fileTransfer.ID] that uniquely identifies the file.
func (ft *ChannelsFileTransfer) Download(fileInfoJSON []byte,
	progressCB FtReceivedProgressCallback, periodMS int) ([]byte, error) {

	cb := func(completed bool, received, total uint16,
		rt channelsFT.ReceivedTransfer, t channelsFT.FilePartTracker, err error) {
		rp := &FtReceivedProgress{
			rt.GetFileID(), completed, int(received), int(total)}
		data, err2 := json.Marshal(rp)
		if err2 != nil {
			jww.FATAL.Panicf("[FT] Failed to JSON marshal %T: %+v", rp, err)
		}
		progressCB.Callback(data, &ChFilePartTracker{t}, err)
	}

	period := time.Duration(periodMS) * time.Millisecond

	fid, err := ft.api.Download(fileInfoJSON, cb, period)
	if err != nil {
		return nil, err
	}

	return fid.Marshal(), nil
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
// Once the download completes, the file will be stored in the event model with
// the given file ID and with the status [channels.ReceptionProcessingComplete].
//
// The [FtReceivedProgressCallback] only indicates the progress of the file
// download, not the status of the file in the event model. You must rely on
// updates from the event model to know when it can be retrieved.
//
// Parameters:
//   - fileIDBytes - Marshalled bytes of the file's [fileTransfer.ID].
//   - progressCB - A callback that reports the progress of the file upload. The
//     callback is called once on initialization, on every progress update (or
//     less if restricted by the period), or on fatal error.
//   - periodMS - A progress callback will be limited from triggering only once
//     per period, in milliseconds.
func (ft *ChannelsFileTransfer) RegisterReceivedProgressCallback(
	fileIDBytes []byte, progressCB FtReceivedProgressCallback,
	periodMS int) error {
	jww.INFO.Printf("[FT] Registering ReceivedProgressCallback to %s.",
		base64.StdEncoding.EncodeToString(fileIDBytes))

	fileID, err := ftCrypto.UnmarshalID(fileIDBytes)
	if err != nil {
		return errors.Errorf("failed to unmarshal file ID: %+v", err)
	}

	cb := func(completed bool, received, total uint16,
		rt channelsFT.ReceivedTransfer, t channelsFT.FilePartTracker, err error) {
		rp := &FtReceivedProgress{
			rt.GetFileID(), completed, int(received), int(total)}
		data, err2 := json.Marshal(rp)
		if err2 != nil {
			jww.FATAL.Panicf("[FT] Failed to JSON marshal %T: %+v", rp, err)
		}
		progressCB.Callback(data, &ChFilePartTracker{t}, err)
	}

	period := time.Duration(periodMS) * time.Millisecond

	return ft.api.RegisterReceivedProgressCallback(fileID, cb, period)
}

////////////////////////////////////////////////////////////////////////////////
// Callbacks                                                                  //
////////////////////////////////////////////////////////////////////////////////

// FtSentProgressCallback contains the method Callback that is called when the
// progress on a sent file changes or an error occurs in the transfer.
//
// The [ChFilePartTracker] can be used to look up the status of individual file
// parts. Note, when completed == true, the [ChFilePartTracker] may be nil.
//
// Any error returned is fatal and the file must either be retried with
// [ChannelsFileTransfer.RetryUpload] or canceled with
// [ChannelsFileTransfer.CloseSend].
//
// This callback only indicates the status of the file transfer, not the status
// of the file in the event model. Do NOT use this callback as an indicator of
// when the file is available in the event model.
//
// Parameters:
//   - payload - JSON of [FtSentProgress], which describes the progress of the
//     current sent transfer.
//   - fpt - File part tracker that allows the lookup of the status of
//     individual file parts.
//   - err - Fatal errors during sending.
type FtSentProgressCallback interface {
	Callback(payload []byte, fpt *ChFilePartTracker, err error)
}

// FtReceivedProgressCallback contains the method Callback that is called when
// the progress on a received file changes or an error occurs in the transfer.
//
// The [ChFilePartTracker] can be used to look up the status of individual file
// parts. Note, when completed == true, the [ChFilePartTracker] may be nil.
//
// This callback only indicates the status of the file transfer, not the status
// of the file in the event model. Do NOT use this callback as an indicator of
// when the file is available in the event model.
//
// Parameters:
//   - payload - JSON of [FtReceivedProgress], which describes the progress of
//     the current received transfer.
//   - fpt - File part tracker that allows the lookup of the status of
//     individual file parts.
//   - err - Fatal errors during receiving.
type FtReceivedProgressCallback interface {
	Callback(payload []byte, fpt *ChFilePartTracker, err error)
}

// FtSentProgress contains the progress information of a sent file transfer.
//
// Example JSON:
//
//	{
//	  "id": "RyJcMqtI3IIM1+YMxRwCcFiOX6AGuIzS+vQaPnqXVT8=",
//	  "completed": false,
//	  "sent": 6,
//	  "received": 145,
//	  "Total": 2048
//	}
type FtSentProgress struct {
	ID        ftCrypto.ID `json:"id"`        // File ID
	Completed bool        `json:"completed"` // True if transfer is successful
	Sent      int         `json:"sent"`      // Number of parts sent
	Received  int         `json:"received"`  // Number of parts received
	Total     int         `json:"total"`     // Total number of file parts
}

// FtReceivedProgress contains the progress information of a received file
// transfer.
//
// Example JSON:
//
//	{
//	  "id": "RyJcMqtI3IIM1+YMxRwCcFiOX6AGuIzS+vQaPnqXVT8=",
//	  "completed": false,
//	  "received": 145,
//	  "Total": 2048
//	}
type FtReceivedProgress struct {
	ID        ftCrypto.ID `json:"id"`        // File ID
	Completed bool        `json:"completed"` // True if transfer is successful
	Received  int         `json:"received"`  // Number of parts received
	Total     int         `json:"total"`     // Total number of file parts
}

////////////////////////////////////////////////////////////////////////////////
// File Part Tracker                                                          //
////////////////////////////////////////////////////////////////////////////////

// ChFilePartTracker contains the channelsFileTransfer.FilePartTracker.
type ChFilePartTracker struct {
	api channelsFT.FilePartTracker
}

// GetPartStatus returns the status of the file part with the given part number.
//
// The possible values for the status are:
//   - 0 < Part does not exist
//   - 0 = unsent
//   - 1 = arrived (sender has sent a part, and it has arrived)
//   - 2 = received (receiver has received a part)
func (fpt ChFilePartTracker) GetPartStatus(partNum int) int {
	return int(fpt.api.GetPartStatus(uint16(partNum)))
}

// GetNumParts returns the total number of file parts in the transfer.
func (fpt ChFilePartTracker) GetNumParts() int {
	return int(fpt.api.GetNumParts())
}
