package edge

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/storage/versioned"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)


const (
	preimageStoreKey     = "preimageStoreKey"
	preimageStoreVersion = 0
)

type Preimage struct{
	Data []byte
	Type string
	Source []byte
}

type Preimages []Preimage

// makes a Preimages object for the given identity and populates
// it with the default preimage for the identity
// does not store to disk
func newPreimages(identity *id.ID) Preimages {
	var pis Preimages
	pis = append(pis, Preimage{
		Data:   identity[:],
		Type:   "default",
		Source: identity[:],
	})
	return pis
}

//loads a Preimages object for the given identity
func loadPreimages(kv *versioned.KV, identity *id.ID)(Preimages, error){

	// Get the data from the kv
	obj, err := kv.Get(preimageKey(identity), preimageStoreVersion)
	if err != nil {
		return nil, errors.WithMessagef(err,"failed to load edge " +
			"Preimages for identity %s", identity)
	}

	var preimageList Preimages

	err = json.Unmarshal(obj.Data, &preimageList)
	if err != nil {
		return nil, errors.WithMessagef(err,"failed to unmarshal edge " +
			"Preimages for identity %s", identity)
	}

	return preimageList, nil
}

//stores the preimage list to disk
func (pis Preimages)save(kv *versioned.KV, identity *id.ID)error{
	//marshal
	data, err := json.Marshal(&pis)
	if err!=nil{
		return errors.WithMessagef(err, "Failed to marshal Preimages list " +
			"for stroage for identity %s", identity)
	}

	// Construct versioning object
	obj := versioned.Object{
		Version:   preimageStoreVersion,
		Timestamp: netTime.Now(),
		Data:      data,
	}

	// Save to storage
	err = kv.Set(preimageKey(identity), preimageStoreVersion, &obj)
	if err != nil {
		return errors.WithMessagef(err, "Failed to store Preimages list " +
			"for identity %s", identity)
	}

	return nil
}

//adds the preimage to the
func (pis Preimages)add(pimg Preimage) Preimages {
	return append(pis,pimg)
}

func (pis Preimages)remove(data []byte) Preimages {

	for i:=0;i<len(pis);i++{
		if bytes.Equal(pis[i].Data,data){
			pis[i] = pis[len(pis)-1]
			return pis[:len(pis)-1]
		}
	}

	return pis
}

func (pis Preimages)delete(kv *versioned.KV, identity *id.ID)error{

	// Save to storage
	err := kv.Delete(preimageKey(identity), preimageStoreVersion)
	if err != nil {
		return errors.WithMessagef(err, "Failed to delete Preimages list " +
			"for identity %s", identity)
	}

	return nil
}

func preimageKey(identity *id.ID)string{
	return fmt.Sprintf("%s:%s",preimageStoreKey,identity)
}


