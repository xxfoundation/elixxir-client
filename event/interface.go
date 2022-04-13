///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package event

import "gitlab.com/elixxir/client/stoppable"

// Callback defines the callback functions for client event reports
type Callback func(priority int, category, evtType, details string)

// Manager reporting api (used internally)
type Manager interface {
	Report(priority int, category, evtType, details string)
	EventService() (stoppable.Stoppable, error)
}
