////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package collective

import (
	"fmt"
	"time"
)

// NewMutate is the constructor of a Mutate object.
func NewMutate(ts time.Time, value []byte, deletion bool) Mutate {
	return Mutate{
		Timestamp: ts.UTC().Unix(),
		Value:     value,
		Deletion:  deletion,
	}
}

// Mutate is the object that is uploaded to a remote service responsible
// for account synchronization. It inherits the private mutate object.
// This prevents recursive calls by json.Marshal on header.MarshalJSON. Any
// changes to the header object fields should be done in header.
type Mutate struct {
	Timestamp int64
	Value     []byte
	Deletion  bool
}

// GetTimestamp returns the timestamp of the mutation in standard go format
// instead of the stored uinx nano count
func (m *Mutate) GetTimestamp() time.Time {
	return time.Unix(0, m.Timestamp)
}

func (m *Mutate) String() string {
	return fmt.Sprintf("ts: %d, deleted? %v, data: %v...",
		m.Timestamp, m.Deletion, m.Value)
}
