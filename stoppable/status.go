////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package stoppable

import (
	"strconv"
)

const (
	Running Status = iota
	Stopping
	Stopped
)

// Status holds the current status of a Stoppable.
type Status uint32

// String prints a string representation of the current Status. This functions
// satisfies the fmt.Stringer interface.
func (s Status) String() string {
	switch s {
	case Running:
		return "running"
	case Stopping:
		return "stopping"
	case Stopped:
		return "stopped"
	default:
		return "INVALID STATUS: " + strconv.FormatUint(uint64(s), 10)
	}
}
