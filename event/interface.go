////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package event

// Callback defines the callback functions for client event reports
type Callback func(priority int, category, evtType, details string)

// Reporter reporting api (used internally)
type Reporter interface {
	Report(priority int, category, evtType, details string)
}
