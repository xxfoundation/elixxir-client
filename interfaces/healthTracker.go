////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package interfaces

type HealthTracker interface {
	AddChannel(chan bool) uint64
	RemoveChannel(uint64)
	AddFunc(f func(bool)) uint64
	RemoveFunc(uint64)
	IsHealthy() bool
	WasHealthy() bool
}
