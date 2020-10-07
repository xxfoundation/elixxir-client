package contact

import (
	"encoding/json"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/bindings"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/primitives/id"
)

// Contact implements the Contact interface defined in interface/contact.go,
// in go, the structure is meant to be edited directly, the functions are for
// bindings compatibility
type Contact struct {
	ID             *id.ID
	DhPubKey       *cyclic.Int
	OwnershipProof []byte
	Facts          []Fact
}

// GetID returns the user ID for this user.
func (c Contact) GetID() []byte {
	return c.ID.Bytes()
}

// GetDHPublicKey returns the public key associated with the Contact.
func (c Contact) GetDHPublicKey() []byte {
	return c.DhPubKey.Bytes()
}

// GetDHPublicKey returns hash of a DH proof of key ownership.
func (c Contact) GetOwnershipProof() []byte {
	return c.OwnershipProof
}

// Returns a fact list for adding and getting facts to and from the contact
func (c Contact) GetFactList() bindings.FactList {
	return FactList{source: &c}
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
