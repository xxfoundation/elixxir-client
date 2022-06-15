////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package bindings

import "gitlab.com/elixxir/client/xxdk"

// e2eTrackerSingleton is used to track E2e objects so that
// they can be referenced by id back over the bindings
var e2eTrackerSingleton = &e2eTracker{
	clients: make(map[int]*E2e),
	count:   0,
}

// E2e BindingsClient wraps the xxdk.E2e, implementing additional functions
// to support the gomobile E2e interface
type E2e struct {
	api *xxdk.E2e
	id  int
}
