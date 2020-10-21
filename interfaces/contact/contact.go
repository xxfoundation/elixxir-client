package contact

import (
	"encoding/json"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/interfaces/bind"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/xx_network/primitives/id"
	"strings"
	jww "github.com/spf13/jwalterweatherman"
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
func (c Contact) GetFactList() bind.FactList {
	return factList{source: &c}
}

// json marshals the contact
func (c Contact) Marshal() ([]byte, error) {
	return json.Marshal(&c)
}

// converts facts to a delineated string with an ending character for transfer
// over the network
func (c Contact) StringifyFacts() string {
	stringList := make([]string, len(c.Facts))
	for index, f := range c.Facts {
		stringList[index] = f.Stringify()
	}

	return strings.Join(stringList, factDelimiter) + factBreak
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

// splits the "facts" portion of the payload from the rest and returns them as
// facts
func UnstringifyFacts(s string) ([]Fact, string, error) {
	parts := strings.SplitN(s, factBreak, 1)
	if len(parts) != 2 {
		return nil, "", errors.New("Invalid fact string passed")
	}
	factStrings := strings.Split(parts[0], factDelimiter)

	var factList []Fact
	for _, fString := range factStrings {
		fact, err := UnstringifyFact(fString)
		if err != nil {
			jww.WARN.Printf("Fact failed to unstringify, dropped: %s",
				err)
		} else {
			factList = append(factList, fact)
		}

	}
	return factList, parts[1], nil
}
