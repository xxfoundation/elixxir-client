////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package connect

import (
	"gitlab.com/elixxir/client/v4/e2e/receive"
	"gitlab.com/xx_network/primitives/netTime"
)

// listenerTracker wraps a listener and updates the last use timestamp on every
// call to Hear.
type listenerTracker struct {
	h *handler
	l receive.Listener
}

// Hear updates the last call timestamp and then calls Hear on the wrapped
// listener.
func (lt *listenerTracker) Hear(item receive.Message) {
	lt.h.updateLastUse(netTime.Now())
	lt.l.Hear(item)
}

// Name returns a name, used for debugging.
func (lt *listenerTracker) Name() string { return lt.l.Name() }
