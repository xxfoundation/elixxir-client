////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package xxdk

import (
	"fmt"
)

// Status holds the status of the network.
type Status int

const (
	// Stopped signifies that the network follower is stopped; none of its
	// processes are running.
	Stopped Status = 0

	// Running signifies that the network follower and its processes are active
	// and running.
	Running Status = 2000

	// Stopping signifies that the network follower has been signalled to stop
	// and is in the processes of stopping the processes.
	Stopping Status = 3000
)

// String returns a human-readable string version of the status. This function
// adheres to the fmt.Stringer interface.
func (s Status) String() string {
	switch s {
	case Stopped:
		return "Stopped"
	case Running:
		return "Running"
	case Stopping:
		return "Stopping"
	default:
		return fmt.Sprintf("Unknown status %d", s)
	}
}
