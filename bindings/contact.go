package bindings

import (
	"encoding/json"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/crypto/xx"
	"gitlab.com/xx_network/primitives/id"
)

type Identity struct {
	ID            []byte
	RSAPrivatePem []byte
	Salt          []byte
	DHKeyPrivate  []byte
}

// MakeIdentity generates a new cryptographic identity for receving
// messages
func (c *Client) MakeIdentity() ([]byte, error) {
	stream := c.api.GetRng().GetStream()
	defer stream.Close()

	//make RSA Key
	rsaKey, err := rsa.GenerateKey(stream,
		rsa.DefaultRSABitLen)
	if err != nil {
		return nil, err
	}

	//make salt
	salt := make([]byte, 32)
	_, err = stream.Read(salt)

	//make dh private key
	privkey := diffieHellman.GeneratePrivateKey(
		len(c.api.GetStorage().GetE2EGroup().GetPBytes()),
		c.api.GetStorage().GetE2EGroup(), stream)

	//make the ID
	id, err := xx.NewID(rsaKey.GetPublic(),
		salt, id.User)
	if err != nil {
		return nil, err
	}

	dhPrivJson, err := privkey.MarshalJSON()
	if err != nil {
		return nil, err
	}

	//create the identity object
	I := Identity{
		ID:            id.Marshal(),
		RSAPrivatePem: rsa.CreatePrivateKeyPem(rsaKey),
		Salt:          salt,
		DHKeyPrivate:  dhPrivJson,
	}

	return json.Marshal(&I)
}

func (c *Client) GetContactFromIdentity(identity []byte) ([]byte, error) {
	uID, _, _, dhKey, err := c.unmarshalIdentity(identity)
	if err != nil {
		return nil, err
	}

	grp := c.api.GetStorage().GetE2EGroup()

	dhPub := grp.ExpG(dhKey, grp.NewInt(1))

	ct := contact.Contact{
		ID:             uID,
		DhPubKey:       dhPub,
		OwnershipProof: nil,
		Facts:          nil,
	}

	return ct.Marshal(), nil
}

func (c *Client) unmarshalIdentity(marshaled []byte) (*id.ID, *rsa.PrivateKey, []byte,
	*cyclic.Int, error) {
	I := Identity{}
	err := json.Unmarshal(marshaled, &I)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	uID, err := id.Unmarshal(I.ID)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	dhkey := c.api.GetStorage().GetE2EGroup().NewInt(1)
	err = dhkey.UnmarshalJSON(I.DHKeyPrivate)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	rsaPriv, err := rsa.LoadPrivateKeyFromPem(I.RSAPrivatePem)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return uID, rsaPriv, I.Salt, dhkey, nil
}

func GetIDFromContact(marshaled []byte) ([]byte, error) {
	cnt, err := contact.Unmarshal(marshaled)
	if err != nil {
		return nil, err
	}

	return cnt.ID.Marshal(), nil
}

func GetPubkeyFromContact(marshaled []byte) ([]byte, error) {
	cnt, err := contact.Unmarshal(marshaled)
	if err != nil {
		return nil, err
	}

	return cnt.ID.Marshal(), nil
}

type Fact struct {
	Fact string
	Type int
}

// SetFactsOnContact replaces the facts on the contact with the passed in facts
// pass in empty facts in order to clear the facts
func SetFactsOnContact(marshaled []byte, facts []byte) ([]byte, error) {
	cnt, err := contact.Unmarshal(marshaled)
	if err != nil {
		return nil, err
	}

	factsList := make([]Fact, 0)
	err = json.Unmarshal(facts, &factsList)
	if err != nil {
		return nil, err
	}

	realFactList := make(fact.FactList, 0, len(factsList))
	for i := range factsList {
		realFactList = append(realFactList, fact.Fact{
			Fact: factsList[i].Fact,
			T:    fact.FactType(factsList[i].Type),
		})
	}

	cnt.Facts = realFactList
	return cnt.Marshal(), nil
}

func GetFactsFromContact(marshaled []byte) ([]byte, error) {
	cnt, err := contact.Unmarshal(marshaled)
	if err != nil {
		return nil, err
	}

	factsList := make([]Fact, len(cnt.Facts))
	for i := range cnt.Facts {
		factsList = append(factsList, Fact{
			Fact: cnt.Facts[i].Fact,
			Type: int(cnt.Facts[i].T),
		})
	}

	factsListMarshaled, err := json.Marshal(&factsList)
	if err != nil {
		return nil, err
	}
	return factsListMarshaled, nil
}
