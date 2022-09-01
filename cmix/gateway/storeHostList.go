////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package gateway

import (
	"bytes"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

// Storage values.
const (
	hostListPrefix  = "hostLists"
	hostListKey     = "hostListIDs"
	hostListVersion = 0
)

// Error messages.
const (
	getStorageErr    = "failed to get host list from storage: %+v"
	unmarshallIdErr  = "unmarshal host list error: %+v"
	unmarshallLenErr = "malformed data: length of data %d incorrect"
)

func saveHostList(kv *versioned.KV, list []*id.ID) error {
	obj := &versioned.Object{
		Version:   hostListVersion,
		Data:      marshalHostList(list),
		Timestamp: netTime.Now(),
	}

	return kv.Set(hostListKey, hostListVersion, obj)
}

// getHostList returns the host list from storage.
func getHostList(kv *versioned.KV) ([]*id.ID, error) {
	obj, err := kv.Get(hostListKey, hostListVersion)
	if err != nil {
		return nil, errors.Errorf(getStorageErr, err)
	}

	return unmarshalHostList(obj.Data)
}

// marshalHostList marshals the list of IDs into a byte slice.
func marshalHostList(list []*id.ID) []byte {
	buff := bytes.NewBuffer(nil)
	buff.Grow(len(list) * id.ArrIDLen)

	for _, hid := range list {
		if hid != nil {
			buff.Write(hid.Marshal())
		} else {
			buff.Write((&id.ID{}).Marshal())
		}
	}

	return buff.Bytes()
}

// unmarshalHostList unmarshal the host list data into an ID list. An error is
// returned if an ID cannot be unmarshalled or if the data is not of the correct
// length.
func unmarshalHostList(data []byte) ([]*id.ID, error) {
	// Return an error if the data is not of the required length
	if len(data)%id.ArrIDLen != 0 {
		return nil, errors.Errorf(unmarshallLenErr, len(data))
	}

	buff := bytes.NewBuffer(data)
	list := make([]*id.ID, 0, len(data)/id.ArrIDLen)

	// Read each ID from data, unmarshal, and add to list
	length := id.ArrIDLen
	for n := buff.Next(length); len(n) == length; n = buff.Next(length) {
		hid, err := id.Unmarshal(n)
		if err != nil {
			return nil, errors.Errorf(unmarshallIdErr, err)
		}

		// If the ID is all zeroes, then treat it as a nil ID.
		if *hid == (id.ID{}) {
			hid = nil
		}

		list = append(list, hid)
	}

	return list, nil
}
