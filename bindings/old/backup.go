////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package old

import (
	"gitlab.com/elixxir/client/backup"
)

type Backup struct {
	b *backup.Backup
}

// UpdateBackupFunc contains a function callback that returns new backups.
type UpdateBackupFunc interface {
	UpdateBackup(encryptedBackup []byte)
}

// InitializeBackup starts the backup processes that returns backup updates when
// they occur. Any time an event occurs that changes the contents of the backup,
// such as adding or deleting a contact, the backup is triggered and an
// encrypted backup is generated and returned on the updateBackupCb callback.
// Call this function only when enabling backup if it has not already been
// initialized or when the user wants to change their password.
// To resume backup process on app recovery, use ResumeBackup.
func InitializeBackup(
	password string, updateBackupCb UpdateBackupFunc, c *Client) (*Backup, error) {
	b, err := backup.InitializeBackup(
		password, updateBackupCb.UpdateBackup, &c.api)
	if err != nil {
		return nil, err
	}

	return &Backup{b}, nil
}

// ResumeBackup starts the backup processes back up with a new callback after it
// has been initialized.
// Call this function only when resuming a backup that has already been
// initialized or to replace the callback.
// To start the backup for the first time or to use a new password, use
// InitializeBackup.
func ResumeBackup(cb UpdateBackupFunc, c *Client) (
	*Backup, error) {
	b, err := backup.ResumeBackup(cb.UpdateBackup, &c.api)
	if err != nil {
		return nil, err
	}

	return &Backup{b}, nil
}

// StopBackup stops the backup processes and deletes the user's password from
// storage. To enable backups again, call InitializeBackup.
func (b *Backup) StopBackup() error {
	return b.b.StopBackup()
}

// IsBackupRunning returns true if the backup has been initialized and is
// running. Returns false if it has been stopped.
func (b *Backup) IsBackupRunning() bool {
	return b.b.IsBackupRunning()
}

// AddJson stores a passed in json string in the backup structure
func (b *Backup) AddJson(json string) {
	b.b.AddJson(json)
}
