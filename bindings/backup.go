///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"encoding/json"
	"gitlab.com/elixxir/client/backup"
	"gitlab.com/xx_network/primitives/id"
)

////////////////////////////////////////////////////////////////////////////////
// Structs and Interfaces                                                     //
////////////////////////////////////////////////////////////////////////////////

// Backup is a bindings-level struct encapsulating the backup.Backup
// client object.
type Backup struct {
	b *backup.Backup
}

// BackupReport is the bindings' representation of the return values of
// NewCmixFromBackup.
//
// Example BackupReport:
//  {
//    "RestoredContacts": [
//      "U4x/lrFkvxuXu59LtHLon1sUhPJSCcnZND6SugndnVID",
//      "15tNdkKbYXoMn58NO6VbDMDWFEyIhTWEGsvgcJsHWAgD"
//    ],
//    "Params": ""
//  }
type BackupReport struct {
	// The list of restored E2E partner IDs
	RestoredContacts []*id.ID

	// The backup parameters found within the backup file
	Params string
}

// UpdateBackupFunc contains a function callback that returns new backups.
type UpdateBackupFunc interface {
	UpdateBackup(encryptedBackup []byte)
}

////////////////////////////////////////////////////////////////////////////////
// Client functions                                                           //
////////////////////////////////////////////////////////////////////////////////

// NewCmixFromBackup initializes a new e2e storage from an encrypted
// backup. Users of this function should delete the storage directory on error.
// Users of this function should call LoadCmix as normal once this call succeeds.
//
// Params
//  - ndfJSON - JSON of the NDF.
//  - storageDir - directory for the storage files.
//  - sessionPassword - password to decrypt the data in the storageDir.
//  - backupPassphrase - backup passphrase provided by the user. Used to decrypt backup.
//  - backupFileContents - the file contents of the backup.
//
// Returns:
//  - []byte - the JSON marshalled bytes of the BackupReport object.
func NewCmixFromBackup(ndfJSON, storageDir, backupPassphrase string,
	sessionPassword, backupFileContents []byte) ([]byte, error) {

	// Restore from backup
	backupIdList, backupParams, err := backup.NewCmixFromBackup(
		ndfJSON, storageDir, backupPassphrase, sessionPassword,
		backupFileContents)
	if err != nil {
		return nil, err
	}

	// Construct report
	report := BackupReport{
		RestoredContacts: backupIdList,
		Params:           backupParams,
	}

	// JSON marshal report
	return json.Marshal(report)

}

////////////////////////////////////////////////////////////////////////////////
// Backup functions                                                           //
////////////////////////////////////////////////////////////////////////////////

// InitializeBackup creates a bindings-layer Backup object.
//
// Params
//  - e2eID - ID of the E2e object in the e2e tracker.
//  - udID - ID of the UserDiscovery object in the ud tracker.
//  - backupPassPhrase - backup passphrase provided by the user. Used to decrypt backup.
//  - cb - the callback to be called when a backup is triggered.
func InitializeBackup(e2eID, udID int, backupPassPhrase string,
	cb UpdateBackupFunc) (*Backup, error) {
	// Retrieve the user from the tracker
	user, err := e2eTrackerSingleton.get(e2eID)
	if err != nil {
		return nil, err
	}

	// Retrieve the UD manager
	ud, err := udTrackerSingleton.get(udID)
	if err != nil {
		return nil, err
	}

	// Initialize backup
	b, err := backup.InitializeBackup(backupPassPhrase, cb.UpdateBackup,
		user.api.GetBackupContainer(), user.api.GetE2E(),
		user.api.GetStorage(), ud.api,
		user.api.GetStorage().GetKV(), user.api.GetRng())
	if err != nil {
		return nil, err
	}

	return &Backup{b: b}, nil
}

// ResumeBackup resumes the backup processes with a new callback.
// Call this function only when resuming a backup that has already been
// initialized or to replace the callback.
// To start the backup for the first time or to use a new password, use
// InitializeBackup.
//
// Params
//  - e2eID - ID of the E2e object in the e2e tracker.
//  - udID - ID of the UserDiscovery object in the ud tracker.
//  - cb - the callback to be called when a backup is triggered.
//    This will replace any callback that has been passed into InitializeBackup.
func ResumeBackup(e2eID, udID int, cb UpdateBackupFunc) (
	*Backup, error) {

	// Retrieve the user from the tracker
	user, err := e2eTrackerSingleton.get(e2eID)
	if err != nil {
		return nil, err
	}

	// Retrieve the UD manager
	ud, err := udTrackerSingleton.get(udID)
	if err != nil {
		return nil, err
	}

	// Resume backup
	b, err := backup.ResumeBackup(cb.UpdateBackup, user.api.GetBackupContainer(),
		user.api.GetE2E(), user.api.GetStorage(), ud.api,
		user.api.GetStorage().GetKV(), user.api.GetRng())
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

// AddJson stores the argument within the Backup structure.
//
// Params
//  - json - JSON string
func (b *Backup) AddJson(json string) {
	b.b.AddJson(json)
}
