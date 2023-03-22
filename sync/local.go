////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package sync

import (
	"encoding/json"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/xx_network/primitives/netTime"
	"strings"
)

const (
	ekvLocalStoreVersion = 0
	ekvLocalStorePrefix  = "sync/LocalKV"
	ekvLocalKeyListKey   = "reserveredKeyList"
)

// EkvLocalStore is a structure adhering to LocalStore. This utilizes
// [versioned.KV] file IO operations.
type EkvLocalStore struct {
	data     *versioned.KV
	keyLists KeyList
}

const (
	notPartOfAListSymbol = ""
)

const (
	invalidKeyErr = "cannot accept key %s with an more than one delimiter (%s)"
)

// NewOrLoadEkvLocalStore is a constructor for EkvLocalStore.
func NewOrLoadEkvLocalStore(kv *versioned.KV) (*EkvLocalStore, error) {
	// Initialize key list structure
	keyLists := make(KeyList, 0)

	// Initialize the non list map
	keyLists[notPartOfAListSymbol] = make(DelimitedList, 0)
	ekvLs := &EkvLocalStore{
		data:     kv.Prefix(ekvLocalStorePrefix),
		keyLists: keyLists,
	}

	return ekvLs, ekvLs.loadKeyList()
}

// Read reads data from path. This will return an error if it fails to read from
// the file path.
//
// This utilizes [ekv.KeyValue] under the hood.
func (ls *EkvLocalStore) Read(path string) ([]byte, error) {
	obj, err := ls.data.Get(path, ekvLocalStoreVersion)
	if err != nil {
		return nil, err
	}
	return obj.Data, nil
}

// Write writes data to the path. This will return an error if it fails to
// write.
//
// This utilizes [ekv.KeyValue] under the hood.
func (ls *EkvLocalStore) Write(key string, value []byte) error {

	if err := ls.updateKeyList(key); err != nil {
		return err
	}

	return ls.data.Set(key, &versioned.Object{
		Version:   ekvLocalStoreVersion,
		Timestamp: netTime.Now(),
		Data:      value,
	})
}

// GetList returns a mapping of all saved keys to the data stored for that key
// in the [versioned.KV].
func (ls *EkvLocalStore) GetList(name string) (KeyValueMap, error) {

	listOfKeys, exists := ls.keyLists[name]
	if !exists {
		return nil, errors.Errorf("could not find list for %s", listOfKeys)
	}

	res := make(KeyValueMap, 0)
	for delimitedKey := range listOfKeys {
		curKey := name + LocalStoreKeyDelimiter + delimitedKey
		data, _ := ls.Read(curKey)
		res[curKey] = data
	}

	return res, nil
}

// updateKeyList is a utility function which will modify and save
// the KeyList when a new key is added to the keyLists.
func (ls *EkvLocalStore) updateKeyList(key string) error {
	keySplit := strings.Split(key, LocalStoreKeyDelimiter)

	// If the key split does not contain the acceptable amount of elements, do
	// not make any state changes in KV, instead return an error
	if len(keySplit) != 1 && len(keySplit) != 2 {
		return errors.Errorf(invalidKeyErr, key, LocalStoreKeyDelimiter)
	}

	// If key does not contain delimiter, add to the non-listed key
	if len(keySplit) == 1 {
		// Dp not perform save operation if the key list already has an entry
		if _, exists := ls.keyLists[notPartOfAListSymbol]; exists {
			return nil
		}

		// Initialize the list
		res := make(map[string]struct{})
		res[notPartOfAListSymbol] = struct{}{}

		// Add the key to the list
		ls.keyLists[notPartOfAListSymbol] = res
	} else {
		listName, delimitedKey := keySplit[0], keySplit[1]

		// If this key has not been added before, initialize the key
		if _, exists := ls.keyLists[listName]; !exists {
			ls.keyLists[listName] = make(map[string]struct{})
		}

		// Dp not perform save operation if the key list already has an entry
		if _, exists := ls.keyLists[listName][delimitedKey]; exists {
			return nil
		}

		// Add the key to the list
		ls.keyLists[listName][delimitedKey] = struct{}{}
	}

	// Save the list if it has been modified
	return ls.saveKeyList()
}

// saveKeyList is a utility function which will save the keyLists to data.
func (ls *EkvLocalStore) saveKeyList() error {
	marshalledKeyList, err := json.Marshal(ls.keyLists)
	if err != nil {
		return err
	}

	return ls.data.Set(ekvLocalKeyListKey, &versioned.Object{
		Version:   ekvLocalStoreVersion,
		Timestamp: netTime.Now(),
		Data:      marshalledKeyList,
	})
}

// loadKeyList loads the keyLists from data and places that into
// EkvLocalStore.keyLists.
func (ls *EkvLocalStore) loadKeyList() error {
	obj, err := ls.data.Get(ekvLocalKeyListKey, ekvLocalStoreVersion)
	if err != nil {
		// If nothing can be loaded, then presume a new list
		return nil
	}
	return json.Unmarshal(obj.Data, &ls.keyLists)
}
