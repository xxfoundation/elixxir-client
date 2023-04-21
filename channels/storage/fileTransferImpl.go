////////////////////////////////////////////////////////////////////////////////
// Copyright © 2023 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package storage

import (
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"time"

	"gitlab.com/elixxir/client/v4/channels"
	cft "gitlab.com/elixxir/client/v4/channelsFileTransfer"
	"gitlab.com/elixxir/crypto/fileTransfer"
)

// ReceiveFile is called when a file upload or download beings.
//
// fileLink and fileData are nillable and may be updated based
// upon the UUID or file ID later.
//
// fileID is always unique to the fileData. fileLink is the JSON of
// channelsFileTransfer.FileLink.
//
// Returns any fatal errors.
func (i *impl) ReceiveFile(fileID fileTransfer.ID, fileLink,
	fileData []byte, timestamp time.Time, status cft.Status) error {

	newFile := &File{
		Id:        fileID.Marshal(),
		Data:      fileData,
		Link:      fileLink,
		Timestamp: timestamp,
		Status:    uint8(status),
	}
	return i.upsertFile(newFile)
}

// UpdateFile is called when a file upload or download completes or changes.
//
// fileLink, fileData, timestamp, and status are all nillable and may be
// updated based upon the file ID at a later date. If a nil value is passed,
// then make no update.
//
// Returns an error if the file cannot be updated. It must return
// channels.NoMessageErr if the file does not exist.
func (i *impl) UpdateFile(fileID fileTransfer.ID, fileLink,
	fileData []byte, timestamp *time.Time, status *cft.Status) error {
	parentErr := "failed to UpdateFile: %+v"

	currentFile := &File{Id: fileID.Marshal()}
	ctx, cancel := newContext()
	err := i.db.WithContext(ctx).Take(currentFile).Error
	cancel()
	if err != nil {
		if errors.Is(gorm.ErrRecordNotFound, err) {
			return errors.Errorf(parentErr, channels.NoMessageErr)
		}
		return errors.Errorf(parentErr, err)
	}

	// Update the fields if specified
	if status != nil {
		currentFile.Status = uint8(*status)
	}
	if timestamp != nil {
		currentFile.Timestamp = *timestamp
	}
	if fileData != nil {
		currentFile.Data = fileData
	}
	if fileLink != nil {
		currentFile.Link = fileLink
	}

	return i.upsertFile(currentFile)
}

// upsertFile is a helper function that will update an existing File
// if File.Id is specified. Otherwise, it will perform an insert.
func (i *impl) upsertFile(newFile *File) error {
	ctx, cancel := newContext()
	err := i.db.WithContext(ctx).Save(newFile).Error
	cancel()
	return err
}

// GetFile returns the ModelFile containing the file data and download link
// for the given file ID.
//
// Returns an error if the file cannot be retrieved. It must return
// channels.NoMessageErr if the file does not exist.
func (i *impl) GetFile(fileID fileTransfer.ID) (
	cft.ModelFile, error) {
	parentErr := "failed to GetFile: %+v"

	resultFile := &File{Id: fileID.Marshal()}
	ctx, cancel := newContext()
	err := i.db.WithContext(ctx).Take(resultFile).Error
	cancel()
	if err != nil {
		if errors.Is(gorm.ErrRecordNotFound, err) {
			return cft.ModelFile{}, channels.NoMessageErr
		}
		return cft.ModelFile{}, errors.Errorf(parentErr, err)
	}

	result := cft.ModelFile{
		ID:        fileTransfer.NewID(resultFile.Data),
		Link:      resultFile.Link,
		Data:      resultFile.Data,
		Timestamp: resultFile.Timestamp,
		Status:    cft.Status(resultFile.Status),
	}
	return result, nil
}

// DeleteFile deletes the file with the given file ID.
//
// Returns fatal errors. It must return channels.NoMessageErr if the file
// does not exist.
func (i *impl) DeleteFile(fileID fileTransfer.ID) error {
	parentErr := "failed to DeleteFile: %+v"

	ctx, cancel := newContext()
	err := i.db.WithContext(ctx).Delete(&File{Id: fileID.Marshal()}).Error
	cancel()

	if err != nil {
		if errors.Is(gorm.ErrRecordNotFound, err) {
			return errors.Errorf(parentErr, channels.NoMessageErr)
		}
		return errors.Errorf(parentErr, err)
	}
	return nil
}
