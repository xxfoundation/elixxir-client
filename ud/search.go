package ud

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/event"
	"gitlab.com/elixxir/client/single"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/factID"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/primitives/id"
)

// SearchTag specifies which callback to trigger when UD receives a search
// request.
const SearchTag = "xxNetwork_UdSearch"

type searchCallback func([]contact.Contact, error)

// Search searches for the passed Facts. The searchCallback will return a list
// of contacts, each having the facts it hit against. This is NOT intended to be
// used to search for multiple users at once; that can have a privacy reduction.
// Instead, it is intended to be used to search for a user where multiple pieces
// of information is known.
func Search(user udE2e,
	udContact contact.Contact, callback searchCallback,
	list fact.FactList,
	params single.RequestParams) ([]id.Round,
	receptionID.EphemeralIdentity, error) {
	jww.INFO.Printf("ud.Search(%s, %s)", list.Stringify(), params.Timeout)

	factHashes, factMap := hashFactList(list)

	// Build the request and marshal it
	request := &SearchSend{Fact: factHashes}
	requestMarshaled, err := proto.Marshal(request)
	if err != nil {
		return []id.Round{}, receptionID.EphemeralIdentity{},
			errors.WithMessage(err, "Failed to form outgoing search request.")
	}

	// Extract information from user
	net := user.GetCmix()
	events := user.GetEventReporter()
	grp := user.GetE2E().GetGroup()
	rng := user.GetRng().GetStream()
	defer rng.Close()

	// Build response handler
	response := searchResponse{
		cb:       callback,
		services: net,
		events:   events,
		grp:      grp,
		factMap:  factMap,
	}

	// Send message
	rndId, ephId, err := single.TransmitRequest(udContact, SearchTag,
		requestMarshaled,
		response, params, net, rng, grp)
	if err != nil {
		return []id.Round{}, receptionID.EphemeralIdentity{},
			errors.WithMessage(err, "Failed to transmit search request.")
	}

	if events != nil {
		events.Report(1, "UserDiscovery", "SearchRequest",
			fmt.Sprintf("Sent: %+v", request))
	}

	return rndId, ephId, err
}

type searchResponse struct {
	cb       searchCallback
	services udCmix
	events   event.Reporter
	grp      *cyclic.Group
	factMap  map[string]fact.Fact
}

func (m searchResponse) Callback(payload []byte,
	receptionID receptionID.EphemeralIdentity,
	round []rounds.Round, err error) {
	if err != nil {
		go m.cb(nil, errors.WithMessage(err, "Failed to search."))
		return
	}

	// Unmarshal the message
	sr := &SearchResponse{}
	if err := proto.Unmarshal(payload, sr); err != nil {
		jww.WARN.Printf("Dropped a search response from user discovery due to "+
			"failed unmarshal: %s", err)
	}

	if m.events != nil {
		m.events.Report(1, "UserDiscovery", "SearchResponse",
			fmt.Sprintf("Received: %+v", sr))
	}

	if sr.Error != "" {
		err = errors.Errorf("User Discovery returned an error on search: %s",
			sr.Error)
		go m.cb(nil, err)
		return
	}

	// return an error if no facts are found
	if len(sr.Contacts) == 0 {
		go m.cb(nil, errors.New("No contacts found in search"))
	}

	c, err := parseContacts(m.grp, sr.Contacts, m.factMap)
	if err != nil {
		go m.cb(nil, errors.WithMessage(err, "Failed to parse contacts from "+
			"remote server."))
		return
	}

	go m.cb(c, nil)
}

// hashFactList hashes each fact in the FactList into a HashFact and returns a
// slice of the HashFacts. Also returns a map of Facts keyed on fact hashes to
// be used for the callback return.
func hashFactList(list fact.FactList) ([]*HashFact, map[string]fact.Fact) {
	hashes := make([]*HashFact, len(list))
	hashMap := make(map[string]fact.Fact, len(list))

	for i, f := range list {
		hashes[i] = &HashFact{
			Hash: factID.Fingerprint(f),
			Type: int32(f.T),
		}
		hashMap[string(factID.Fingerprint(f))] = f
	}

	return hashes, hashMap
}

// parseContacts parses the list of Contacts in the SearchResponse and returns a
// list of contact.Contact with their ID and public key.
func parseContacts(grp *cyclic.Group, response []*Contact,
	hashMap map[string]fact.Fact) ([]contact.Contact, error) {
	contacts := make([]contact.Contact, len(response))
	// Convert each contact message into a new contact.Contact
	for i, c := range response {
		// Unmarshal user ID bytes
		uid, err := id.Unmarshal(c.UserID)
		if err != nil {
			return nil, errors.Errorf("failed to parse Contact user ID: %+v", err)
		}
		var facts []fact.Fact
		if c.Username != "" {
			facts = []fact.Fact{{c.Username, fact.Username}}
		}
		// Create new Contact
		contacts[i] = contact.Contact{
			ID:       uid,
			DhPubKey: grp.NewIntFromBytes(c.PubKey),
			Facts:    facts,
		}

		// Assign each Fact with a matching hash to the Contact
		for _, hashFact := range c.TrigFacts {
			if f, exists := hashMap[string(hashFact.Hash)]; exists {
				contacts[i].Facts = append(contacts[i].Facts, f)
			}
		}
	}

	return contacts, nil
}
