////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package collective

import (
	"encoding/json"
	"time"
)

// NewMutate is the constructor of a Mutate object.
func NewMutate(ts time.Time, value []byte, deletion bool) Mutate {
	return Mutate{
		Timestamp: ts.UTC().UnixNano(),
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

// MarshalJSON adheres to json.Marshaler.
func (m *Mutate) MarshalJSON() ([]byte, error) {
	// Marshal the current mutate
	return json.Marshal(*m)
}

// UnmarshalJSON adheres to json.Unmarshaler.
func (m *Mutate) UnmarshalJSON(data []byte) error {
	// Unmarshal mutate
	tx := Mutate{}
	if err := json.Unmarshal(data, &tx); err != nil {
		return err
	}

	*m = Mutate(tx)
	return nil
}

// GetTimestamp returns the timestamp of the mutation in standard go format
// instead of the stored uinx nano count
func (m *Mutate) GetTimestamp() time.Time {
	return time.Unix(0, m.Timestamp)
}
