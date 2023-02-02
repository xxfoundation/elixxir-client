////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package sync

import (
	"gitlab.com/xx_network/primitives/utils"
	"time"
)

type FilesystemRemoteStorage struct{}

func (f *FilesystemRemoteStorage) Read(path string) ([]byte, error) {
	return utils.ReadFile(path)
}

func (f *FilesystemRemoteStorage) Write(path string, data []byte) error {
	return utils.WriteFileDef(path, data)
}

func (f *FilesystemRemoteStorage) GetLastModified(path string) (time.Time, error) {
	return utils.GetLastModified(path)
}
