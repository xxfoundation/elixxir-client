package edge

import (
	"encoding/base64"
	"encoding/json"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/interfaces/preimage"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

const (
	preimageStoreKey     = "preimageStoreKey"
	preimageStoreVersion = 0
)

type Preimage struct {
	Data   []byte
	Type   string
	Source []byte
}

// key returns the key used to identify the Preimage in a map.
func (pi Preimage) key() string {
	return base64.StdEncoding.EncodeToString(pi.Data)
}

// Preimages is a map of unique Preimage keyed on their Data.
type Preimages map[string]Preimage

// newPreimages makes a Preimages object for the given identity and populates
// it with the default preimage for the identity. Does not store to disk.
func newPreimages(identity *id.ID) Preimages {
	defaultPreimage := Preimage{
		Data:   preimage.MakeDefault(identity),
		Type:   preimage.Default,
		Source: identity[:],
	}
	pis := Preimages{
		defaultPreimage.key(): defaultPreimage,
	}

	return pis
}

// add adds the preimage to the list.
func (pis Preimages) add(preimage Preimage) bool {
	if _, exists := pis[preimage.key()]; exists {
		return false
	}

	pis[preimage.key()] = preimage

	return true
}

// remove deletes the Preimage with the matching data from the list.
func (pis Preimages) remove(data []byte) {
	key := base64.StdEncoding.EncodeToString(data)
	delete(pis, key)
}

////////////////////////////////////////////////////////////////////////////////
// Storage Functions                                                          //
////////////////////////////////////////////////////////////////////////////////

// loadPreimages loads a Preimages object for the given identity.
func loadPreimages(kv *versioned.KV, identity *id.ID) (Preimages, error) {

	// get the data from storage
	obj, err := kv.Get(preimagesKey(identity), preimageStoreVersion)
	if err != nil {
		return nil, errors.WithMessagef(err, "Failed to load edge Preimages "+
			"for identity %s", identity)
	}

	var preimageList Preimages
	err = json.Unmarshal(obj.Data, &preimageList)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to unmarshal edge "+
			"Preimages for identity %s", identity)
	}

	return preimageList, nil
}

// save stores the preimage list to disk.
func (pis Preimages) save(kv *versioned.KV, identity *id.ID) error {
	// JSON marshal
	data, err := json.Marshal(&pis)
	if err != nil {
		return errors.WithMessagef(err, "Failed to marshal Preimages list "+
			"for stroage for identity %s", identity)
	}

	// Construct versioning object
	obj := versioned.Object{
		Version:   preimageStoreVersion,
		Timestamp: netTime.Now(),
		Data:      data,
	}

	// Save to storage
	err = kv.Set(preimagesKey(identity), preimageStoreVersion, &obj)
	if err != nil {
		return errors.WithMessagef(err, "Failed to store Preimages list for "+
			"identity %s", identity)
	}

	return nil
}

// delete removes the Preimages from storage.
func (pis Preimages) delete(kv *versioned.KV, identity *id.ID) error {
	err := kv.Delete(preimagesKey(identity), preimageStoreVersion)
	if err != nil {
		return errors.WithMessagef(err, "Failed to delete Preimages list for "+
			"identity %s", identity)
	}

	return nil
}

// preimagesKey generates the key for saving a Preimages to storage.
func preimagesKey(identity *id.ID) string {
	return preimageStoreKey + ":" + identity.String()
}
