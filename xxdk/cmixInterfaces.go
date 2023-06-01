////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package xxdk

type ClientError struct {
	Source  string
	Message string
	Trace   string
}

type ClientErrorReport func(source, message, trace string)

type HealthTracker interface {
	AddChannel(chan bool) uint64
	RemoveChannel(uint64)
	AddFunc(f func(bool)) uint64
	RemoveFunc(uint64)
	IsHealthy() bool
	WasHealthy() bool
}
