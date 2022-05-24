////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package e2e

const (
	defaultNotifyUponCompletion = true
)

// Params contains parameters used for E2E file transfer.
type Params struct {
	// NotifyUponCompletion indicates if a final notification message is sent
	// to the recipient on completion of file transfer. If true, the ping is
	NotifyUponCompletion bool
}

// DefaultParams returns a Params object filled with the default values.
func DefaultParams() Params {
	return Params{
		NotifyUponCompletion: defaultNotifyUponCompletion,
	}
}
