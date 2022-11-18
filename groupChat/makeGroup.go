////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package groupChat

import (
	"strconv"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	gs "gitlab.com/elixxir/client/v4/groupChat/groupStore"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/crypto/group"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
)

// Error messages.
const (
	maxInitMsgSizeErr = "new group request message length %d > %d maximum size"
	getPrivKeyErr     = "failed to get private key from partner: %+v"
	minMembersErr     = "length of membership list %d < %d minimum allowed"
	maxMembersErr     = "length of membership list %d > %d maximum allowed"
	getPartnerErr     = "failed to get partner %s: %+v"
	makeMembershipErr = "failed to assemble group chat membership: %+v"
	newIdPreimageErr  = "failed to create group ID preimage: %+v"
	newKeyPreimageErr = "failed to create group key preimage: %+v"
)

// MaxInitMessageSize is the maximum allowable length of the initial message
// sent in a group request.
const MaxInitMessageSize = 256

// RequestStatus signals the status of the group requests on group creation.
type RequestStatus int

const (
	NotSent     RequestStatus = iota // Error occurred before sending requests
	AllFail                          // Sending of all requests failed
	PartialSent                      // Sending of some request failed
	AllSent                          // Sending of all request succeeded
)

// MakeGroup sends groupChat requests to all members over an authenticated
// channel. The leader of a groupChat must have an authenticated channel with
// each member of the groupChat to add them to the groupChat. It blocks until
// all the groupChat requests are sent. Returns an error if at least one request
// to a member fails to send.
func (m *manager) MakeGroup(membership []*id.ID, name, msg []byte) (gs.Group,
	[]id.Round, RequestStatus, error) {
	// Return an error if the message is too long
	if len(msg) > MaxInitMessageSize {
		return gs.Group{}, nil, NotSent,
			errors.Errorf(maxInitMsgSizeErr, len(msg), MaxInitMessageSize)
	}

	// Build membership and DH key list from list of IDs
	mem, dkl, err := m.buildMembership(membership)
	if err != nil {
		return gs.Group{}, nil, NotSent, err
	}

	// Generate ID and key preimages
	idPreimage, keyPreimage, err := getPreimages(m.getRng())
	if err != nil {
		return gs.Group{}, nil, NotSent, err
	}

	// Create new group ID and key
	groupID := group.NewID(idPreimage, mem)
	groupKey := group.NewKey(keyPreimage, mem)

	// Generate group creation timestamp stripped of the monotonic clock
	created := netTime.Now().Round(0)

	// Create new group and add to manager
	g := gs.NewGroup(name, groupID, groupKey, idPreimage,
		keyPreimage, msg, created, mem, dkl)

	jww.INFO.Printf("[GC] Created new group %q with ID %s and %d members %s",
		g.Name, g.ID, len(g.Members), g.Members)

	// Send all group requests
	roundIDs, status, err := m.sendRequests(g)
	if err == nil {
		err = m.JoinGroup(g)
	}

	return g, roundIDs, status, err
}

// buildMembership retrieves the contact object for each member ID and creates a
// new membership from them. The caller is set as the leader. For a member to be
// added, the group leader must have an authenticated channel with the member.
func (m *manager) buildMembership(members []*id.ID) (group.Membership,
	gs.DhKeyList, error) {
	// Return an error if the membership list has too few or too many members
	if len(members) < group.MinParticipants {
		return nil, nil,
			errors.Errorf(minMembersErr, len(members), group.MinParticipants)
	} else if len(members) > group.MaxParticipants {
		return nil, nil,
			errors.Errorf(maxMembersErr, len(members), group.MaxParticipants)
	}

	dkl := make(gs.DhKeyList, len(members))

	// Lookup partner contact objects from their ID
	contacts := make([]contact.Contact, len(members))
	var err error
	for i, uid := range members {
		partner, err := m.getE2eHandler().GetPartner(uid)
		if err != nil {
			return nil, nil, errors.Errorf(getPartnerErr, uid, err)
		}

		contacts[i] = contact.Contact{
			ID:       partner.PartnerId(),
			DhPubKey: partner.PartnerRootPublicKey(),
		}

		dkl.Add(partner.MyRootPrivateKey(), group.Member{
			ID:    partner.PartnerId(),
			DhKey: partner.PartnerRootPublicKey(),
		}, m.getE2eGroup())
	}

	// Create new Membership from contact list and client's own contact.
	user := m.gs.GetUser()
	leader := contact.Contact{ID: user.ID, DhPubKey: user.DhKey}
	mem, err := group.NewMembership(leader, contacts...)
	if err != nil {
		return nil, nil, errors.Errorf(makeMembershipErr, err)
	}

	return mem, dkl, nil
}

// getPreimages generates and returns the group ID preimage and the group key
// preimage. This function allows the stream to
func getPreimages(streamGen *fastRNG.StreamGenerator) (group.IdPreimage,
	group.KeyPreimage, error) {

	// get new stream and defer its close
	rng := streamGen.GetStream()
	defer rng.Close()

	idPreimage, err := group.NewIdPreimage(rng)
	if err != nil {
		return group.IdPreimage{}, group.KeyPreimage{},
			errors.Errorf(newIdPreimageErr, err)
	}

	keyPreimage, err := group.NewKeyPreimage(rng)
	if err != nil {
		return group.IdPreimage{}, group.KeyPreimage{},
			errors.Errorf(newKeyPreimageErr, err)
	}

	return idPreimage, keyPreimage, nil
}

// String prints the description of the status code. This functions satisfies
// the fmt.Stringer interface.
func (rs RequestStatus) String() string {
	switch rs {
	case NotSent:
		return "NotSent"
	case AllFail:
		return "AllFail"
	case PartialSent:
		return "PartialSent"
	case AllSent:
		return "AllSent"
	default:
		return "INVALID STATUS"
	}
}

// Message prints a full description of the status code.
func (rs RequestStatus) Message() string {
	switch rs {
	case NotSent:
		return "an error occurred before sending any group requests"
	case AllFail:
		return "all group requests failed to send"
	case PartialSent:
		return "some group requests failed to send"
	case AllSent:
		return "all groups requests successfully sent"
	default:
		return "INVALID STATUS " + strconv.Itoa(int(rs))
	}
}
