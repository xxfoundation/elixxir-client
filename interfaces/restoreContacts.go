////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package interfaces

// RestoreContactsUpdater interface provides a callback function
// for receiving update information from RestoreContactsFromBackup.
type RestoreContactsUpdater interface {
	// RestoreContactsCallback is called to report the current # of contacts
	// that have been found and how many have been restored
	// against the total number that need to be
	// processed. If an error occurs it it set on the err variable as a
	// plain string.
	RestoreContactsCallback(numFound, numRestored, total int, err string)
}
