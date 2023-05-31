////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package versioned

import (
	"reflect"
	"testing"
	"time"
)

// Shows that all fields can be serialized/deserialized correctly using json
func TestVersionedObject_MarshalUnmarshal(t *testing.T) {
	original := Object{
		Version:   8,
		Timestamp: time.Date(1, 2, 3, 4, 5, 6, 7, time.UTC),
		Data:      []byte("original text"),
	}

	marshalled := original.Marshal()

	unmarshalled := Object{}
	err := unmarshalled.Unmarshal(marshalled)
	if err != nil {
		// Should never happen
		t.Fatal(err)
	}

	if !reflect.DeepEqual(original, unmarshalled) {
		t.Error("Original and deserialized objects not equal")
	}
	t.Logf("%+v", unmarshalled)
}
