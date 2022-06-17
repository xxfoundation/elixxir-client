///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package xxdk

import (
	"fmt"
)

type Status int

const (
	Stopped  Status = 0
	Running  Status = 2000
	Stopping Status = 3000
)

func (s Status) String() string {
	switch s {
	case Stopped:
		return "Stopped"
	case Running:
		return "Running"
	case Stopping:
		return "Stopping"
	default:
		return fmt.Sprintf("Unknown state %d", s)
	}
}
