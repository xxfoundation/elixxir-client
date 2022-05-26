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

// Example contact.Contact:
// {"ID":"emV6aW1hAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD",  // User ID (base64)
//  // DH Public key
//  "DhPubKey": {"Value":5897856983236448841349236507987372549159552598715284610195378196051089368134573280466076038189672561458883294904902687846973238178822842233241860785766185816826285758360932391225518001053367819933965095474045876471502521426634463472280885866375056810295845912596776232660230744646657460711296721451092117848644200333013666241751893383151321460171904648894952567100917586408052597859524654581795844734383581139432945085822846793496012536562496524464153486043864392759112173594825221043729881090076151145795436552852351725727608490246482831755459144881209180199576801001999800444715698088783576119852633280838566256264123448817581650501316117908817592489368379136557137873340250734512199521378972799693841244968719164574541767128546852594550026407820738695953499407931441091576816112217698417722574592750265071234831071430050448725796700241505489986766630847142868559624597459165280989719389157750815411870806046789780648398173408766005145891531993469827100942349,
//               "Fingerprint":16801541511233098363},
//  // Ownership proof for this contact
//  "OwnershipProof":"Mjp8KAn7wK/VYYR2BOlG57a9Zh3HA/wHM8R6RnBdGnNCXMR5Mel9ESSYv3g/6b6RXKqTcDHDyd4aaP6g/Ju+dQ==",
//  // List of associated facts
//  "Facts":[{"Fact":"zezima","T":0}]}

// Identity struct
// Example marshalled Identity:
// {"ID":"emV6aW1hAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD",  // User ID (base64)
//  // RSA Private key (PEM format)
//  "RSAPrivatePem":"LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFcFFJQkFBS0NBUUVBNU15dTdhYjBJOS9UL1BFUUxtd2x3ejZHV3FjMUNYemVIVXhoVEc4bmg1WWRWSXMxCmJ2THpBVjNOMDJxdXN6K2s4TVFEWjBtejMzdkswUmhPczZIY0NUSFdzTEpXRkE5WWpzWWlCRi9qTDd1bmd1ckIKL2tvK1JJSnNrWGFWaEZaazRGdERoRXhTNWY4RnR0Qmk1NmNLZmdJQlVKT3ozZi9qQllTMkxzMlJ6cWV5YXM3SApjV2RaME9TclBTT3BiYlViU1FPbS9LWnlweGZHU21yZ2oxRUZuU1dZZ2xGZTdUOTRPbHF5MG14QTV5clVXbHorCk9sK3hHbXpCNUp4WUFSMU9oMFQrQTk4RWMrTUZHNm43L1MraDdzRDgybGRnVnJmbStFTzRCdmFKeTRESGZGMWgKNnp6QnVnY25NUVFGc0dLeDFYWC9COTVMdUpPVjdyeXlDbzZGbHdJREFRQUJBb0lCQVFDaUh6OGNlcDZvQk9RTAphUzBVRitHeU5VMnlVcVRNTWtTWThoUkh1c09CMmFheXoybHZVb3RLUHBPbjZRSWRWVTJrcE4vY2dtY0lSb2x5CkhBMDRUOHJBWVNaRlVqaVlRajkzKzRFREpJYXd2Z0YyVEs1bFoyb3oxVTdreStncU82V0RMR2Z0Q0wvODVQWEIKa210aXhnUXpRV3g1RWcvemtHdm03eURBalQxeDloNytsRjJwNFlBam5kT2xTS0dmQjFZeTR1RXBQd0kwc1lWdgpKQWc0MEFxbllZUmt4emJPbmQxWGNjdEJFN2Z1VDdrWXhoeSs3WXYrUTJwVy9BYmh6NGlHOEY1MW9GMGZwV0czCmlISDhsVXZFTkp2SUZEVHZ0UEpESlFZalBRN3lUbGlGZUdrMXZUQkcyQkpQNExzVzhpbDZOeUFuRktaY1hOQ24KeHVCendiSlJBb0dCQVBUK0dGTVJGRHRHZVl6NmwzZmg3UjJ0MlhrMysvUmpvR3BDUWREWDhYNERqR1pVd1RGVQpOS2tQTTNjS29ia2RBYlBDb3FpL0tOOVBibk9QVlZ3R3JkSE9vSnNibFVHYmJGamFTUzJQMFZnNUVhTC9rT2dUCmxMMUdoVFpIUWk1VUlMM0p4M1Z3T0ZRQ3RQOU1UQlQ0UEQvcEFLbDg3VTJXN3JTY1dGV1ZGbFNkQW9HQkFPOFUKVmhHWkRpVGFKTWVtSGZIdVYrNmtzaUlsam9aUVVzeGpmTGNMZ2NjV2RmTHBqS0ZWTzJNN3NqcEJEZ0w4NmFnegorVk14ZkQzZ1l0SmNWN01aMVcwNlZ6TlNVTHh3a1dRY1hXUWdDaXc5elpyYlhCUmZRNUVjMFBlblVoWWVwVzF5CkpkTC8rSlpQeDJxSzVrQytiWU5EdmxlNWdpcjlDSGVzTlR5enVyckRBb0dCQUl0cTJnN1RaazhCSVFUUVNrZ24Kb3BkRUtzRW4wZExXcXlBdENtVTlyaWpHL2l2eHlXczMveXZDQWNpWm5VVEp0QUZISHVlbXVTeXplQ2g5QmRkegoyWkRPNUdqQVBxVHlQS3NudFlNZkY4UDczZ1NES1VSWWVFbHFDejdET0c5QzRzcitPK3FoN1B3cCtqUmFoK1ZiCkNuWllNMDlBVDQ3YStJYUJmbWRkaXpLbEFvR0JBSmo1dkRDNmJIQnNISWlhNUNJL1RZaG5YWXUzMkVCYytQM0sKMHF3VThzOCtzZTNpUHBla2Y4RjVHd3RuUU4zc2tsMk1GQWFGYldmeVFZazBpUEVTb0p1cGJzNXA1enNNRkJ1bwpncUZrVnQ0RUZhRDJweTVwM2tQbDJsZjhlZXVwWkZScGE0WmRQdVIrMjZ4eWYrNEJhdlZJeld3NFNPL1V4Q3crCnhqbTNEczRkQW9HQWREL0VOa1BjU004c1BCM3JSWW9MQ2twcUV2U0MzbVZSbjNJd3c1WFAwcDRRVndhRmR1ckMKYUhtSE1EekNrNEUvb0haQVhFdGZ2S2tRaUI4MXVYM2c1aVo4amdYUVhXUHRteTVIcVVhcWJYUTlENkxWc3B0egpKL3R4SWJLMXp5c1o2bk9IY1VoUUwyVVF6SlBBRThZNDdjYzVzTThEN3kwZjJ0QURTQUZNMmN3PQotLS0tLUVORCBSU0EgUFJJVkFURSBLRVktLS0tLQ==",
//  // Salt for identity (base64)
//  "Salt":"4kk02v0NIcGtlobZ/xkxqWz8uH/ams/gjvQm14QT0dI=",
//  // DH Private key
//  "DHKeyPrivate":"eyJWYWx1ZSI6NDU2MDgzOTEzMjA0OTIyODA5Njg2MDI3MzQ0MzM3OTA0MzAyODYwMjM2NDk2NDM5NDI4NTcxMTMwNDMzOTQwMzgyMTIyMjY4OTQzNTMyMjIyMzc1MTkzNTEzMjU4MjA4MDA0NTczMDY4MjEwNzg2NDI5NjA1MjA0OTA3MjI2ODI5OTc3NTczMDkxODY0NTY3NDExMDExNjQxNCwiRmluZ2VycHJpbnQiOjE2ODAxNTQxNTExMjMzMDk4MzYzfQ=="
// }
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

// GetContactFromIdentity accepts a marshalled Identity object and returns a marshalled contact.Contact object
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
	err = dhkey.UnmarshalJSON([]byte(I.DHKeyPrivate))
	if err != nil {
		return nil, nil, nil, nil, err
	}

	rsaPriv, err := rsa.LoadPrivateKeyFromPem([]byte(I.RSAPrivatePem))
	if err != nil {
		return nil, nil, nil, nil, err
	}

	return uID, rsaPriv, I.Salt, dhkey, nil
}

// GetIDFromContact accepts a marshalled contact.Contact object & returns a marshalled id.ID object
func GetIDFromContact(marshaled []byte) ([]byte, error) {
	cnt, err := contact.Unmarshal(marshaled)
	if err != nil {
		return nil, err
	}

	return cnt.ID.Marshal(), nil
}

// GetPubkeyFromContact accepts a marshalled contact.Contact object & returns a json marshalled large.Int DhPubKey
func GetPubkeyFromContact(marshaled []byte) ([]byte, error) {
	cnt, err := contact.Unmarshal(marshaled)
	if err != nil {
		return nil, err
	}

	return json.Marshal(cnt.DhPubKey)
}

// TODO: this seems completely pointless, as the FactList type is effectively the same thing
type Fact struct {
	Fact string
	Type int
}

// SetFactsOnContact replaces the facts on the contact with the passed in facts
// pass in empty facts in order to clear the facts
// Accepts a marshalled contact.Contact object & a marshalled list of Fact objects
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

// GetFactsFromContact accepts a marshalled contact.Contact object, returning its marshalled list of Fact objects
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
