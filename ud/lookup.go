package ud

import (
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/client/single"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

// LookupTag specifies which callback to trigger when UD receives a lookup
// request.
const LookupTag = "xxNetwork_UdLookup"

// TODO: reconsider where this comes from
const maxLookupMessages = 20

// Lookup returns the public key of the passed ID as known by the user discovery
// system or returns by the timeout.
func Lookup(udContact contact.Contact,
	services cmix.Client,
	callback single.Response,
	rng *fastRNG.StreamGenerator,
	uid *id.ID, grp *cyclic.Group,
	timeout time.Duration) error {

	jww.INFO.Printf("ud.Lookup(%s, %s)", uid, timeout)
	return lookup(services, callback, rng, uid, grp, timeout, udContact)
}

// BatchLookup performs a Lookup operation on a list of user IDs.
// The lookup performs a callback on each lookup on the returned contact object
// constructed from the response.
func BatchLookup(udContact contact.Contact,
	services cmix.Client, callback single.Response,
	rng *fastRNG.StreamGenerator,
	uids []*id.ID, grp *cyclic.Group,
	timeout time.Duration) {
	jww.INFO.Printf("ud.BatchLookup(%s, %s)", uids, timeout)

	for _, uid := range uids {
		go func(localUid *id.ID) {
			err := lookup(services, callback, rng, localUid, grp, timeout, udContact)
			if err != nil {
				jww.WARN.Printf("Failed batch lookup on user %s: %v", localUid, err)
			}
		}(uid)
	}

	return
}

// lookup is a helper function which sends a lookup request to the user discovery
// service. It will construct a contact object off of the returned public key.
// The callback will be called on that contact object.
func lookup(services cmix.Client, callback single.Response,
	rng *fastRNG.StreamGenerator,
	uid *id.ID, grp *cyclic.Group,
	timeout time.Duration, udContact contact.Contact) error {
	// Build the request and marshal it
	request := &LookupSend{UserID: uid.Marshal()}
	requestMarshaled, err := proto.Marshal(request)
	if err != nil {
		return errors.WithMessage(err,
			"Failed to form outgoing lookup request.")
	}

	// todo: figure out callback structure, maybe you do not pass
	//  in a single.Response but a manager callback?
	f := func(payload []byte, receptionID receptionID.EphemeralIdentity,
		round rounds.Round, err error) {
		m.lookupResponseProcess(payload, receptionID, round, err)
	}

	p := single.RequestParams{
		Timeout:     timeout,
		MaxMessages: maxLookupMessages,
		CmixParam:   cmix.GetDefaultCMIXParams(),
	}

	stream := rng.GetStream()
	defer stream.Close()

	rndId, ephId, err := single.TransmitRequest(udContact, LookupTag, requestMarshaled,
		callback, p, services, stream,
		grp)
	if err != nil {
		return errors.WithMessage(err, "Failed to transmit lookup request.")
	}

	return nil
}

// lookupResponseProcess processes the lookup response. The returned public key
// and the user ID will be constructed into a contact object. The contact object
// will be passed into the callback.
func (m *Manager) lookupResponseProcess(uid *id.ID, cb single.Response,
	payload []byte, err error) {
	if err != nil {
		go cb.Callback(contact.Contact{}, errors.WithMessage(err, "Failed to lookup."))
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
