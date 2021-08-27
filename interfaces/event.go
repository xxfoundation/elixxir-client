///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package interfaces

// EventCallbackFunction defines the callback functions for client event reports
type EventCallbackFunction func(priority int, category, evtType, details string)

// EventManager reporting api (used internally)
type EventManager interface {
	Report(priority int, category, evtType, details string)
}
