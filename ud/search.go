package ud

import (
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/factID"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

// SearchTag specifies which callback to trigger when UD receives a search
// request.
const SearchTag = "xxNetwork_UdSearch"

// TODO: reconsider where this comes from
const maxSearchMessages = 20

type searchCallback func([]contact.Contact, error)

// Search searches for the passed Facts. The searchCallback will return a list
// of contacts, each having the facts it hit against. This is NOT intended to be
// used to search for multiple users at once; that can have a privacy reduction.
// Instead, it is intended to be used to search for a user where multiple pieces
// of information is known.
func (m *Manager) Search(list fact.FactList, callback searchCallback, timeout time.Duration) error {
	jww.INFO.Printf("ud.Search(%s, %s)", list.Stringify(), timeout)

	factHashes, factMap := hashFactList(list)

	// Build the request and marshal it
	request := &SearchSend{Fact: factHashes}
	requestMarshaled, err := proto.Marshal(request)
	if err != nil {
		return errors.WithMessage(err, "Failed to form outgoing search request.")
	}

	f := func(payload []byte, err error) {
		m.searchResponseHandler(factMap, callback, payload, err)
	}

	// get UD contact
	c, err := m.getContact()
	if err != nil {
		return err
	}

	err = m.single.TransmitSingleUse(c, requestMarshaled, SearchTag,
		maxSearchMessages, f, timeout)
	if err != nil {
		return errors.WithMessage(err, "Failed to transmit search request.")
	}

	if m.client != nil {
		m.client.ReportEvent(1, "UserDiscovery", "SearchRequest",
			fmt.Sprintf("Sent: %+v", request))
	}

	return nil
}

func (m *Manager) searchResponseHandler(factMap map[string]fact.Fact,
	callback searchCallback, payload []byte, err error) {

	if err != nil {
		go callback(nil, errors.WithMessage(err, "Failed to search."))
		return
	}

	// Unmarshal the message
	searchResponse := &SearchResponse{}
	if err := proto.Unmarshal(payload, searchResponse); err != nil {
		jww.WARN.Printf("Dropped a search response from user discovery due to "+
			"failed unmarshal: %s", err)
	}

	if m.client != nil {
		m.client.ReportEvent(1, "UserDiscovery", "SearchResponse",
			fmt.Sprintf("Received: %+v", searchResponse))
	}

	if searchResponse.Error != "" {
		err = errors.Errorf("User Discovery returned an error on search: %s",
			searchResponse.Error)
		go callback(nil, err)
		return
	}

	// return an error if no facts are found
	if len(searchResponse.Contacts) == 0 {
		go callback(nil, errors.New("No contacts found in search"))
	}

	c, err := m.parseContacts(searchResponse.Contacts, factMap)
	if err != nil {
		go callback(nil, errors.WithMessage(err, "Failed to parse contacts from "+
			"remote server."))
		return
	}

	go callback(c, nil)
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
func (m *Manager) parseContacts(response []*Contact,
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
			DhPubKey: m.grp.NewIntFromBytes(c.PubKey),
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
