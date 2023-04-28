package versioned

import (
	"errors"
	jww "github.com/spf13/jwalterweatherman"
	"strconv"
	"strings"
)

const (
	// map handler constants
	MapDesignator     = "🗺️"
	MapKeysListSuffix = "_🗺️MapKeys"
	//MapKeysListFmt     = "%s" + MapKeysListSuffix
	MapElementKeySuffix = "_🗺️MapElement"
	//MapElementKeyFmt   = "%s:%s_%d" + MapElementKeySuffix
	UsedMapDesignatorError = "cannot use \"" + MapDesignator + "\" in a key " +
		"name, it is reserved"
)

type KeyType uint8

const (
	KeyTypeNormal KeyType = iota
	KeyTypeMapFile
	KeyTypeMapElement
)

// IsValidKey checks if a map name or element name
// is allowable by now allowing the designator character '🗺️'
func IsValidKey(str string) error {
	if strings.Contains(str, MapDesignator) {
		return errors.New(UsedMapDesignatorError)
	}
	return nil
}

// MakeMapKey creates the storage key for a map
func MakeMapKey(mapName string) string {
	return mapName + MapKeysListSuffix
}

// GetMapName returns the map name, will garble a non mapKey
// use DetectMap
func GetMapName(mapKey string) string {
	suffixTrim := len(mapKey) - len(MapKeysListSuffix)
	return mapKey[:suffixTrim]
}

// GetElementName returns the element name, will garble a non mapKey
func GetElementName(key string) (mapName, elementName string) {
	suffixTrim := len(key) - len(MapElementKeySuffix)
	key = key[:suffixTrim]

	//get the length of the element name
	var elementKeyLenStr string
	key, elementKeyLenStr = splitLast(key, "_")
	elementKeyLen, err := strconv.Atoi(elementKeyLenStr)
	if err != nil {
		jww.FATAL.Panicf("Failed to get map name length from the "+
			"key: %+v", err)
	}
	// -1 is to get rid of the ':'
	mapName = key[:len(key)-elementKeyLen-1]
	elementName = key[len(key)-elementKeyLen:]
	return mapName, elementName
}

// MakeElementKey creates the storage key for an element in a map
func MakeElementKey(mapName, elementName string) string {
	return mapName + ":" + elementName + "_" + strconv.Itoa(len(elementName)) + MapElementKeySuffix
}

// DetectMap Detects either a map or an element and its map
func DetectMap(key string) (result KeyType, mapName, elementName string) {
	if strings.HasSuffix(key, MapKeysListSuffix) {
		mapName = GetMapName(key)
		return KeyTypeMapFile, mapName, ""
	} else if strings.HasSuffix(key, MapElementKeySuffix) {
		mapName, elementName = GetElementName(key)
		return KeyTypeMapElement, mapName, elementName
	}
	return KeyTypeNormal, "", ""
}

// IsMapElement detects if the key is a mapElement
func IsMapElement(key string) (IsMapElement bool, mapName, elementName string) {
	if strings.HasSuffix(key, MapElementKeySuffix) {
		mapName, elementName = GetElementName(key)
		return true, mapName, elementName
	}
	return false, "", ""
}

func splitLast(str string, sep string) (before, after string) {
	var i int
	for i = len(str) - 1; i >= 0; i-- {
		if str[i] == sep[0] {
			break
		}
	}

	return str[:i], str[i+1:]
}
