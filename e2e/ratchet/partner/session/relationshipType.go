////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package session

import (
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
)

type RelationshipType uint

const (
	Send RelationshipType = iota
	Receive
)

func (rt RelationshipType) String() string {
	switch rt {
	case Send:
		return "Send"
	case Receive:
		return "Receive"
	default:
		return fmt.Sprintf("Unknown relationship type: %d", rt)
	}
}

func (rt RelationshipType) Prefix() string {
	switch rt {
	case Send:
		return "Send"
	case Receive:
		return "Receive"
	default:
		jww.FATAL.Panicf("No Prefix for relationship type: %s", rt)
	}
	return ""
}
