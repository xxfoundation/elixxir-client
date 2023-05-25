package collective

import (
	"encoding/json"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/ekv"
)

var ErrMapInconsistent = errors.New("element is in the map file but not" +
	"in the ekv")
var ErrMapElementNotFound = errors.New("element is not in the map")

const errWrap = "for map '%s' element '%s'"

// StoreMapElement stores a versioned map element into the KV. This relies
// on the underlying remote [KV.StoreMapElement] function to lock and control
// updates, but it uses [versioned.Object] values.
// All Map storage functions update the remote.
func (r *internalKV) StoreMapElement(mapName, element string,
	value []byte) error {
	elementsMap := make(map[string][]byte)
	elementsMap[element] = value
	_, err := r.txLog.WriteMap(mapName, elementsMap, nil)
	return err
}

// DeleteMapElement deletes a versioned map element from a KV. This relies
// on the underlying remote [KV.StoreMapElement] function to lock and control
// updates, but it uses [versioned.Object] values.
// All Map storage functions update the remote.
func (r *internalKV) DeleteMapElement(mapName, element string) ([]byte, error) {
	elementsMap := make(map[string]struct{})
	elementsMap[element] = struct{}{}
	old, err := r.txLog.WriteMap(mapName, nil, elementsMap)
	if err != nil {
		return nil, err
	}

	oldData, exists := old[element]
	if !exists {
		return nil, nil
	}
	return oldData, nil
}

// StoreMap saves each element of the map, then updates the map structure
// and deletes no longer used keys in the map.
// All Map storage functions update the remote.
func (r *internalKV) StoreMap(mapName string, value map[string][]byte) error {
	_, err := r.txLog.WriteMap(mapName, value, nil)
	return err
}

// GetMapElement looks up the element for the given map
func (r *internalKV) GetMapElement(mapName, elementName string) ([]byte, error) {
	mapKey := versioned.MakeMapKey(mapName)
	elementKey := versioned.MakeElementKey(mapName, elementName)
	keys := []string{elementKey, mapKey}

	var old []byte
	var existed bool

	op := func(files map[string]ekv.Operable, _ ekv.Extender) error {
		mapFile := files[mapKey]
		mapFileBytes, _ := mapFile.Get()
		mapSet, err := getMapFile(mapFileBytes, len(old))
		if err != nil {
			return err
		}

		if mapSet.Has(elementName) {
			elementFile := files[elementKey]
			old, existed = elementFile.Get()
			if !existed {
				return errors.Wrapf(ErrMapInconsistent, errWrap, mapName,
					elementName)
			}
		} else {
			return errors.Wrapf(ErrMapElementNotFound, errWrap, mapName,
				elementName)
		}

		return nil
	}

	err := r.kv.Transaction(op, keys...)
	if err != nil {
		return nil, err
	}

	return old, nil
}

// GetMap get an entire map from disk
func (r *internalKV) GetMap(mapName string) (map[string][]byte, error) {
	mapKey := versioned.MakeMapKey(mapName)

	var mapMap map[string][]byte

	op := func(files map[string]ekv.Operable, ext ekv.Extender) error {

		// get the mapSet
		mapFile := files[mapKey]
		mapFileBytes, _ := mapFile.Get()
		mapSet, err := getMapFile(mapFileBytes, 10)
		if err != nil {
			return err
		}

		// create the keys to lookup with from the mapSet
		mapKeys := make([]string, 0, mapSet.Length())
		keysToNames := make(map[string]string, mapSet.Length())
		mapMap = make(map[string][]byte, mapSet.Length())

		for elementName := range mapSet {
			elementKey := versioned.MakeElementKey(mapName, elementName)
			mapKeys = append(mapKeys, elementKey)
			keysToNames[elementKey] = elementName
		}

		// Get the keys
		elementFiles, err := ext.Extend(mapKeys)
		if err != nil {
			return err
		}

		//convert the files to the return map
		var exists bool
		for elementKey, elementFile := range elementFiles {
			elementName := keysToNames[elementKey]
			mapMap[elementName], exists = elementFile.Get()
			if !exists {
				jww.WARN.Printf("%+v", errors.Wrapf(ErrMapInconsistent,
					errWrap, mapName, elementName))
			}
		}

		return nil
	}

	err := r.kv.Transaction(op, mapKey)
	if err != nil {
		return nil, err
	}

	return mapMap, nil
}

func getMapFile(data []byte, length int) (set, error) {
	mapFile := newSet(uint(length))
	if data != nil {
		err := json.Unmarshal(data, &mapFile)
		if err != nil {
			return nil, err
		}
	}
	return mapFile, nil
}

// set object to allow for easier implementation of map operations
type set map[string]struct{}

func newSet(size uint) set {
	if size == 0 {
		return make(set)
	} else {
		return make(set, size)
	}
}

func (ks set) Has(element string) bool {
	_, ok := ks[element]
	return ok
}

func (ks set) Add(element string) {
	ks[element] = struct{}{}
}

func (ks set) Get() []string {
	list := make([]string, 0, len(ks))

	for key := range ks {
		list = append(list, key)
	}
	return list
}

func (ks set) Delete(element string) {
	delete(ks, element)
}

func (ks set) Length() int {
	return len(ks)
}
