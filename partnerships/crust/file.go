////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package crust

// BackupFile is the structure that will be used in UploadBackup. It contains
// all the data necessary to format the file into an HTTP POST request.
type BackupFile struct {
	Data []byte
	Path string
}

// NewBackupFile is the constructor for the BackupFile object.
func NewBackupFile(filePath string, fileData []byte) BackupFile {
	return BackupFile{
		Data: fileData,
		Path: filePath,
	}
}
