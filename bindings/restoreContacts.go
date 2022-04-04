///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package bindings

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/xxmutils"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/xx_network/primitives/id"
)

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

// RestoreContactsReport is a gomobile friendly report structure
// for determining which IDs restored, which failed, and why.
type RestoreContactsReport struct {
	restored []*id.ID
	failed   []*id.ID
	errs     []error
	restErr  error
}

// LenRestored returns the length of ID's restored.
func (r *RestoreContactsReport) LenRestored() int {
	return len(r.restored)
}

// LenFailed returns the length of the ID's failed.
func (r *RestoreContactsReport) LenFailed() int {
	return len(r.failed)
}

// GetRestoredAt returns the restored ID at index
func (r *RestoreContactsReport) GetRestoredAt(index int) []byte {
	return r.restored[index].Bytes()
}

// GetFailedAt returns the failed ID at index
func (r *RestoreContactsReport) GetFailedAt(index int) []byte {
	return r.failed[index].Bytes()
}

// GetErrorAt returns the error string at index
func (r *RestoreContactsReport) GetErrorAt(index int) string {
	return r.errs[index].Error()
}

// GetRestoreContactsError returns an error string. Empty if no error.
func (r *RestoreContactsReport) GetRestoreContactsError() string {
	if r.restErr == nil {
		return ""
	}
	return r.restErr.Error()
}

// RestoreContactsFromBackup takes as input the jason output of the
// `NewClientFromBackup` function, unmarshals it into IDs, looks up
// each ID in user discovery, and initiates a session reset request.
// This function will not return until every id in the list has been sent a
// request. It should be called again and again until it completes.
// xxDK users should not use this function. This function is used by
// the mobile phone apps and are not intended to be part of the xxDK. It
// should be treated as internal functions specific to the phone apps.
func RestoreContactsFromBackup(backupPartnerIDs []byte, client *Client,
	udManager *UserDiscovery, lookupCB LookupCallback,
	updatesCb RestoreContactsUpdater) *RestoreContactsReport {

	extLookupCB := func(c contact.Contact, myErr error) {
		jww.INFO.Printf("extLookupCB triggered: %v, %v", c, myErr)
		bindingsContact := &Contact{c: &c}
		errStr := ""
		if myErr != nil {
			jww.WARN.Printf("restore err on lookup: %+v",
				myErr)
			errStr = myErr.Error()
		}
		if lookupCB != nil {
			lookupCB.Callback(bindingsContact, errStr)
		}
	}

	restored, failed, errs, err := xxmutils.RestoreContactsFromBackup(
		backupPartnerIDs, &client.api, udManager.ud, extLookupCB,
		updatesCb)

	return &RestoreContactsReport{
		restored: restored,
		failed:   failed,
		errs:     errs,
		restErr:  err,
	}

}
