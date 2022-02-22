package ud

import (
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

// LookupTag specifies which callback to trigger when UD receives a lookup
// request.
const LookupTag = "xxNetwork_UdLookup"

// TODO: reconsider where this comes from
const maxLookupMessages = 20

type lookupCallback func(contact.Contact, error)

// MultiLookupCallback returns the result of many paralel lookups
type multiLookupCallback func(c contact.Contact, index int, uid *id.ID, err error)

type lookupResponse struct {
	C     contact.Contact
	err   error
	index int
	id    *id.ID
}

// Lookup returns the public key of the passed ID as known by the user discovery
// system or returns by the timeout.
func (m *Manager) Lookup(uid *id.ID, callback lookupCallback, timeout time.Duration) error {
	jww.INFO.Printf("ud.Lookup(%s, %s)", uid, timeout)
	return m.lookup(uid, callback, timeout)
}

// MultiLookup performs a Lookup operation on a list of user IDs.
// The lookup performs a callback on each lookup on the returned contact object
// constructed from the response.
func (m *Manager) MultiLookup(idList []*id.ID, callback multiLookupCallback,
	timeout time.Duration) {
	jww.INFO.Printf("ud.MultiLookup(%s, %s)", idList, timeout)

	//loop through the IDs and send the lookup
	for index, uid := range idList {
		cb := func(c contact.Contact, err error) {
			callback(c, index, uid, err)
		}

		go func(localId *id.ID) {
			err := m.Lookup(localId, cb, timeout)
			if err != nil {
				jww.WARN.Printf("Failed batch lookup on user %s: %v", localId, err)
			}
		}(uid)
	}

	return
}

// lookup is a helper function which sends a lookup request to the user discovery
// service. It will construct a contact object off of the returned public key.
// The callback will be called on that contact object.
func (m *Manager) lookup(uid *id.ID, callback lookupCallback, timeout time.Duration) error {
	// Build the request and marshal it
	request := &LookupSend{UserID: uid.Marshal()}
	requestMarshaled, err := proto.Marshal(request)
	if err != nil {
		return errors.WithMessage(err, "Failed to form outgoing lookup request.")
	}

	f := func(payload []byte, err error) {
		m.lookupResponseProcess(uid, callback, payload, err)
	}

	// Get UD contact
	c, err := m.getContact()
	if err != nil {
		return err
	}

	err = m.single.TransmitSingleUse(c, requestMarshaled, LookupTag,
		maxLookupMessages, f, timeout)
	if err != nil {
		return errors.WithMessage(err, "Failed to transmit lookup request.")
	}

	return nil
}

// lookupResponseProcess processes the lookup response. The returned public key
// and the user ID will be constructed into a contact object. The contact object
// will be passed into the callback.
func (m *Manager) lookupResponseProcess(uid *id.ID, callback lookupCallback,
	payload []byte, err error) {
	if err != nil {
		go callback(contact.Contact{}, errors.WithMessage(err, "Failed to lookup."))
		return
	}

	// Unmarshal the message
	lookupResponse := &LookupResponse{}
	if err := proto.Unmarshal(payload, lookupResponse); err != nil {
		jww.WARN.Printf("Dropped a lookup response from user discovery due to "+
			"failed unmarshal: %s", err)
	}
	if lookupResponse.Error != "" {
		err = errors.Errorf("User Discovery returned an error on lookup: %s",
			lookupResponse.Error)
		go callback(contact.Contact{}, err)
		return
	}

	c := contact.Contact{
		ID:       uid,
		DhPubKey: m.grp.NewIntFromBytes(lookupResponse.PubKey),
	}

	if lookupResponse.Username != "" {
		c.Facts = fact.FactList{{
			Fact: lookupResponse.Username,
			T:    fact.Username,
		}}
	}

	go callback(c, nil)
}
