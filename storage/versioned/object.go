////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package versioned

import (
	"encoding/json"
	"fmt"
	"time"
)

// Object is used by VersionedKeyValue to keep track of
// versioning and time of storage
type Object struct {
	// Used to determine version Upgrade, if any
	Version uint64

	// Set when this object is written
	Timestamp time.Time

	// Serialized version of original object
	Data []byte
}

// Unmarshal deserializes a Object from a byte slice. It's used to
// make these storable in a KeyValue.
// Object exports all fields and they have simple types, so
// json.Unmarshal works fine.
func (v *Object) Unmarshal(data []byte) error {
	return json.Unmarshal(data, v)
}

// Marshal serializes a Object into a byte slice. It's used to
// make these storable in a KeyValue.
// Object exports all fields and they have simple types, so
// json.Marshal works fine.
func (v *Object) Marshal() []byte {
	d, err := json.Marshal(v)
	// Not being to marshal this simple object means something is really
	// wrong
	if err != nil {
		panic(fmt.Sprintf("Could not marshal: %+v", v))
	}
	return d
}
