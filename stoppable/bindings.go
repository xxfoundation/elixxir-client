///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package stoppable

import "time"

type Bindings interface {
	Close(timeoutMS int) error
	IsRunning() bool
	Name() string
}

func WrapForBindings(s Stoppable) Bindings {
	return &bindingsStoppable{s: s}
}

type bindingsStoppable struct {
	s Stoppable
}

func (bs *bindingsStoppable) Close(timeoutMS int) error {
	timeout := time.Duration(timeoutMS) * time.Millisecond
	return bs.s.Close(timeout)
}

func (bs *bindingsStoppable) IsRunning() bool {
	return bs.s.IsRunning()
}

func (bs *bindingsStoppable) Name() string {
	return bs.s.Name()
}
