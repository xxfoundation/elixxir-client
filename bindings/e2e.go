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

// Login creates and returns a new E2e object
// and adds it to the e2eTrackerSingleton
// identity can be left nil such that a new
// TransmissionIdentity will be created automatically
func (e *E2e) Login(cmixId int, identity []byte) (*E2e, error) {
	cmix, err := cmixTrackerSingleton.get(cmixId)
	if err != nil {
		return nil, err
	}

	newIdentity := &xxdk.TransmissionIdentity{}
	if identity == nil {
		newIdentity = nil
	} else {
		newIdentity, err = cmix.unmarshalIdentity(identity)
		if err != nil {
			return nil, err
		}
	}

	newE2e, err := xxdk.Login(cmix.api, nil, newIdentity)
	if err != nil {
		return nil, err
	}
	return e2eTrackerSingleton.make(newE2e), nil
}
