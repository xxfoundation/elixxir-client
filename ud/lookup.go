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
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

// LookupTag specifies which callback to trigger when UD receives a lookup
// request.
const LookupTag = "xxNetwork_UdLookup"

// TODO: reconsider where this comes from
const maxLookupMessages = 20

type lookupCallback func(contact.Contact, error)

// Lookup returns the public key of the passed ID as known by the user discovery
// system or returns by the timeout.
func Lookup(services CMix,
	rng csprng.Source, grp *cyclic.Group,
	udContact contact.Contact, callback lookupCallback,
	uid *id.ID, timeout time.Duration) (id.Round,
	receptionID.EphemeralIdentity, error) {

	jww.INFO.Printf("ud.Lookup(%s, %s)", uid, timeout)
	return lookup(services, rng, uid, grp, timeout, udContact, callback)
}

// BatchLookup performs a Lookup operation on a list of user IDs.
// The lookup performs a callback on each lookup on the returned contact object
// constructed from the response.
func BatchLookup(udContact contact.Contact,
	services CMix, callback lookupCallback,
	rng csprng.Source,
	uids []*id.ID, grp *cyclic.Group,
	timeout time.Duration) {
	jww.INFO.Printf("ud.BatchLookup(%s, %s)", uids, timeout)

	for _, uid := range uids {
		go func(localUid *id.ID) {
			_, _, err := lookup(services, rng, localUid, grp,
				timeout, udContact, callback)
			if err != nil {
				jww.WARN.Printf("Failed batch lookup on user %s: %v",
					localUid, err)
			}
		}(uid)
	}

	return
}

// lookup is a helper function which sends a lookup request to the user discovery
// service. It will construct a contact object off of the returned public key.
// The callback will be called on that contact object.
func lookup(net CMix,
	rng csprng.Source,
	uid *id.ID, grp *cyclic.Group,
	timeout time.Duration, udContact contact.Contact,
	callback lookupCallback) ([]id.Round,
	receptionID.EphemeralIdentity, error) {
	// Build the request and marshal it
	request := &LookupSend{UserID: uid.Marshal()}
	requestMarshaled, err := proto.Marshal(request)
	if err != nil {
		return id.Round(0),
			receptionID.EphemeralIdentity{}, errors.WithMessage(err,
				"Failed to form outgoing lookup request.")
	}

	response := lookupResponse{
		cb: callback,
	}

	p := single.RequestParams{
		Timeout:             timeout,
		MaxResponseMessages: maxLookupMessages,
		CmixParam:           cmix.GetDefaultCMIXParams(),
	}

	return single.TransmitRequest(udContact, LookupTag, requestMarshaled,
		response, p, net, rng,
		grp)
}

// lookupResponse processes the lookup response. The returned public key
// and the user ID will be constructed into a contact object. The contact object
// will be passed into the callback.
type lookupResponse struct {
	cb  lookupCallback
	uid *id.ID
	grp *cyclic.Group
}

func (m lookupResponse) Callback(payload []byte,
	receptionID receptionID.EphemeralIdentity,
	round rounds.Round, err error) {

	if err != nil {
		go m.cb(contact.Contact{}, errors.WithMessage(err, "Failed to lookup."))
		return
	}

	// Unmarshal the message
	lr := &LookupResponse{}
	if err := proto.Unmarshal(payload, lr); err != nil {
		jww.WARN.Printf("Dropped a lookup response from user discovery due to "+
			"failed unmarshal: %s", err)
	}
	if lr.Error != "" {
		err = errors.Errorf("User Discovery returned an error on lookup: %s",
			lr.Error)
		go m.cb(contact.Contact{}, err)
		return
	}

	c := contact.Contact{
		ID:       m.uid,
		DhPubKey: m.grp.NewIntFromBytes(lr.PubKey),
	}

	if lr.Username != "" {
		c.Facts = fact.FactList{{
			Fact: lr.Username,
			T:    fact.Username,
		}}
	}

	go m.cb(c, nil)
}
