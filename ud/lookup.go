////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package ud

import (
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v5/cmix/identity/receptionID"
	"gitlab.com/elixxir/client/v5/cmix/rounds"
	"gitlab.com/elixxir/client/v5/single"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
)

// LookupTag specifies which callback to trigger when UD receives a lookup
// request.
const LookupTag = "xxNetwork_UdLookup"

type lookupCallback func(contact.Contact, error)

// Lookup returns the public key of the passed ID as known by the user discovery
// system or returns by the timeout.
func Lookup(user udE2e,
	udContact contact.Contact, callback lookupCallback,
	uid *id.ID, p single.RequestParams) ([]id.Round,
	receptionID.EphemeralIdentity, error) {

	// Extract information from user
	net := user.GetCmix()
	grp := user.GetE2E().GetGroup()
	rng := user.GetRng().GetStream()
	defer rng.Close()

	jww.INFO.Printf("ud.Lookup(%s, %s)", uid, p.Timeout)
	return lookup(net, rng, uid, grp, udContact, callback, p)
}

// lookup is a helper function which sends a lookup request to the user discovery
// service. It will construct a contact object off of the returned public key.
// The callback will be called on that contact object.
func lookup(net udCmix, rng csprng.Source, uid *id.ID,
	grp *cyclic.Group, udContact contact.Contact,
	callback lookupCallback,
	p single.RequestParams) (
	[]id.Round, receptionID.EphemeralIdentity, error) {
	// Build the request and marshal it
	request := &LookupSend{UserID: uid.Marshal()}
	requestMarshaled, err := proto.Marshal(request)
	if err != nil {
		return []id.Round{},
			receptionID.EphemeralIdentity{}, errors.WithMessage(err,
				"Failed to form outgoing lookup request.")
	}

	response := lookupResponse{
		cb:  callback,
		uid: uid,
		grp: grp,
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
	rounds []rounds.Round, err error) {

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
