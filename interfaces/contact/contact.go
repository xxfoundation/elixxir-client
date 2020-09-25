package contact

import (
	"encoding/json"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/interfaces"
	"gitlab.com/elixxir/client/interfaces/contact/fact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/primitives/id"
)

// Contact implements the Contact interface defined in interface/contact.go,
// in go, the structure is meant to be edited directly, the functions are for
// bindings compatibility
type Contact struct {
	ID       *id.ID
	DhPubKey *cyclic.Int
	Facts    []fact.Fact
}

// GetID returns the user ID for this user.
func (c Contact) GetID() []byte {
	return c.ID.Bytes()
}

// GetPublicKey returns the publickey bytes for this user.
func (c Contact) GetDHPublicKey() []byte {
	return c.DhPubKey.Bytes()
}

// Adds a fact to the contact. Because the contact is pass by value, this makes
// a new copy with the fact
func (c Contact) AddFact(f interfaces.Fact) interfaces.Contact {
	fNew := fact.Fact{
		Fact: f.Get(),
		T:    fact.Type(f.GetType()),
	}
	c.Facts = append(c.Facts, fNew)
	return c
}

func (c Contact) NumFacts() int {
	return len(c.Facts)
}

func (c Contact) GetFact(i int) (interfaces.Fact, error) {
	if i >= len(c.Facts) || i < 0 {
		return nil, errors.Errorf("Cannot get a a fact at position %v, "+
			"only %v facts", i, len(c.Facts))
	}
	return c.Facts[i], nil
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
