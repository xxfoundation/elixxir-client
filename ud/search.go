package ud

import (
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/interfaces/contact"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/comms/network/dataStructures"
	"gitlab.com/elixxir/crypto/factID"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

type searchCallback func([]contact.Contact, error)

// Search...
func (m *Manager) Search(list fact.FactList, callback searchCallback, timeout time.Duration) error {
	// Get the ID of this comm so it can be connected to its response
	commID := m.getCommID()

	factHashes, factMap := hashFactList(list)

	// Build the request
	request := &SearchSend{
		Fact:   factHashes,
		CommID: commID,
	}

	requestMarshaled, err := proto.Marshal(request)
	if err != nil {
		return errors.WithMessage(err, "Failed to form outgoing search request")
	}

	msg := message.Send{
		Recipient:   m.udID,
		Payload:     requestMarshaled,
		MessageType: message.UdSearch,
	}

	// Register the request in the response map so it can be processed on return
	responseChan := make(chan *SearchResponse)
	m.inProgressSearchMux.Lock()
	m.inProgressSearch[commID] = responseChan
	m.inProgressSearchMux.Unlock()

	// Send the request
	rounds, _, err := m.net.SendE2E(msg, params.GetDefaultE2E())
	if err != nil {
		return errors.WithMessage(err, "Failed to send the search request")
	}

	// Register the round event to capture if the round fails
	roundFailChan := make(chan dataStructures.EventReturn, len(rounds))

	for _, round := range rounds {
		// Subtract a millisecond to ensure this timeout will trigger before the
		// one below
		m.net.GetInstance().GetRoundEvents().AddRoundEventChan(round,
			roundFailChan, timeout-1*time.Millisecond, states.FAILED)
	}

	// Start the go routine which will trigger the callback
	go func() {
		timer := time.NewTimer(timeout)

		var err error
		var c []contact.Contact

		select {
		// Return an error if the round fails
		case <-roundFailChan:
			err = errors.New("One or more rounds failed to resolve; " +
				"search not delivered")

		// Return an error if the timeout is reached
		case <-timer.C:
			err = errors.New("Response from User Discovery did not come " +
				"before timeout")

		// Return the contacts if one is returned
		case response := <-responseChan:
			if response.Error != "" {
				err = errors.Errorf("User Discovery returned an error on "+
					"search: %s", response.Error)
			} else {
				c, err = m.parseContacts(response.Contacts, factMap)
			}
		}

		// Delete the response channel from the map
		m.inProgressSearchMux.Lock()
		delete(m.inProgressSearch, commID)
		m.inProgressSearchMux.Unlock()

		// Call the callback last in case it is blocking
		callback(c, err)
	}()

	return nil
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
func (m *Manager) parseContacts(response []*Contact, hashMap map[string]fact.Fact) ([]contact.Contact, error) {
	contacts := make([]contact.Contact, len(response))

	// Convert each contact message into a new contact.Contact
	for i, c := range response {
		// Unmarshal user ID bytes
		uid, err := id.Unmarshal(c.UserID)
		if err != nil {
			return nil, errors.Errorf("Failed to parse Contact user ID: %+v", err)
		}

		// Create new Contact
		contacts[i] = contact.Contact{
			ID:       uid,
			DhPubKey: m.grp.NewIntFromBytes(c.PubKey),
			Facts:    []fact.Fact{},
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
