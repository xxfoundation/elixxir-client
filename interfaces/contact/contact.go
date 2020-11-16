package contact

import (
	"encoding/json"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/primitives/id"
)

const factDelimiter = ","
const factBreak = ";"


// Contact implements the Contact interface defined in interface/contact.go,
// in go, the structure is meant to be edited directly, the functions are for
// bindings compatibility
type Contact struct {
	ID             *id.ID
	DhPubKey       *cyclic.Int
	OwnershipProof []byte
	Facts          fact.FactList
}

func (c Contact) Marshal() ([]byte, error) {
	return json.Marshal(&c)
}

func Unmarshal(b []byte) (Contact, error) {
	c := Contact{}
	err := json.Unmarshal(b, &c)
	if err != nil {
		return c, err
	}
	for i, fact := range c.Facts {
		if !fact.T.IsValid() {
			return Contact{}, errors.Errorf("Fact %v/%v has invalid "+
				"type: %s", i, len(c.Facts), fact.T)
		}
	}
	return c, nil
}