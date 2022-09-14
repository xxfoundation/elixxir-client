////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package store

import (
	"github.com/cloudflare/circl/dh/sidh"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"

	"gitlab.com/xx_network/primitives/id"

	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/primitives/format"
)

func (s *Store) AddReceivedLegacySIDH(c contact.Contact, key *sidh.PublicKey,
	round rounds.Round) error {
	s.mux.Lock()
	defer s.mux.Unlock()
	jww.DEBUG.Printf("AddReceived new contact: %s, prefix: %s",
		c.ID, s.kv.GetPrefix())

	if _, ok := s.receivedByIDLegacySIDH[*c.ID]; ok {
		return errors.Errorf("Cannot add contact for partner "+
			"%s, one already exists", c.ID)
	}
	if _, ok := s.sentByID[*c.ID]; ok {
		return errors.Errorf("Cannot add contact for partner "+
			"%s, one already exists", c.ID)
	}
	r := newReceivedRequestLegacySIDH(s.kv, c, key, round)

	s.receivedByIDLegacySIDH[*r.GetContact().ID] = r
	if err := s.save(); err != nil {
		jww.FATAL.Panicf("Failed to save Sent Request Map after adding "+
			"partner %s", c.ID)
	}

	return nil
}

// GetAllReceivedRequests returns a slice of all recieved requests.
func (s *Store) GetAllReceivedRequestsLegacySIDH() []*ReceivedRequestLegacySIDH {

	s.mux.RLock()
	rr := make([]*ReceivedRequestLegacySIDH, 0, len(s.receivedByIDLegacySIDH))

	for _, r := range s.receivedByIDLegacySIDH {
		rr = append(rr, r)
	}
	s.mux.RUnlock()

	return rr
}

func (s *Store) AddSentLegacySIDH(partner *id.ID, partnerHistoricalPubKey, myPrivKey,
	myPubKey *cyclic.Int, sidHPrivA *sidh.PrivateKey,
	sidHPubA *sidh.PublicKey, fp format.Fingerprint,
	reset bool) (*SentRequest, error) {
	s.mux.Lock()
	defer s.mux.Unlock()

	if !reset {
		if sentRq, ok := s.sentByID[*partner]; ok {
			return sentRq, errors.Errorf("sent request "+
				"already exists for partner %s",
				partner)
		}
		if _, ok := s.receivedByIDLegacySIDH[*partner]; ok {
			return nil, errors.Errorf("received request "+
				"already exists for partner %s",
				partner)
		}
	}

	sr, err := newSentRequest(s.kv, partner, partnerHistoricalPubKey,
		myPrivKey, myPubKey, sidHPrivA, sidHPubA, fp, reset)

	if err != nil {
		return nil, err
	}

	s.sentByID[*sr.GetPartner()] = sr
	s.srh.Add(sr)
	if err = s.save(); err != nil {
		jww.FATAL.Panicf("Failed to save Sent Request Map after "+
			"adding partner %s", partner)
	}

	return sr, nil
}
