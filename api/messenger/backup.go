////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package messenger

import "sync"

type TriggerBackup func(reason string)

// Container contains the trigger to call to initiate a backup.
type Container struct {
	triggerBackup TriggerBackup
	mux           sync.RWMutex
}

// TriggerBackup triggers a backup if a backup trigger has been set.
// The passed in reason will be printed to the log when the backup is sent. It
// should be in the paste tense. For example, if a contact is deleted, the
// reason can be "contact deleted" and the log will show:
//	Triggering backup: contact deleted
func (bc *Container) TriggerBackup(reason string) {
	bc.mux.RLock()
	defer bc.mux.RUnlock()
	if bc.triggerBackup != nil {
		bc.triggerBackup(reason)
	}
}

// SetBackup sets the backup trigger function which will cause a backup to start
// on the next event that triggers is.
func (bc *Container) SetBackup(triggerBackup TriggerBackup) {
	bc.mux.Lock()
	defer bc.mux.Unlock()

	bc.triggerBackup = triggerBackup
}
