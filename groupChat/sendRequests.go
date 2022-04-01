///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package groupChat

import (
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/e2e"
	gs "gitlab.com/elixxir/client/groupChat/groupStore"
	"gitlab.com/elixxir/crypto/group"
	"gitlab.com/xx_network/primitives/id"
	"strings"
)

// Error messages.
const (
	resendGroupIdErr      = "cannot resend request to nonexistent group with ID %s"
	protoMarshalErr       = "failed to form outgoing group chat request: %+v"
	sendE2eErr            = "failed to send group request via E2E to member %s: %+v"
	sendRequestAllErr     = "failed to send all %d group request messages: %s"
	sendRequestPartialErr = "failed to send %d/%d group request messages: %s"
)

// ResendRequest allows a groupChat request to be sent again.
func (m Manager) ResendRequest(groupID *id.ID) ([]id.Round, RequestStatus, error) {
	g, exists := m.gs.Get(groupID)
	if !exists {
		return nil, NotSent, errors.Errorf(resendGroupIdErr, groupID)
	}

	jww.DEBUG.Printf("Resending group requests for group %s.", groupID)

	return m.sendRequests(g)
}

// sendRequests sends group requests to each member in the group except for the
// leader/sender
func (m Manager) sendRequests(g gs.Group) ([]id.Round, RequestStatus, error) {
	// Build request message
	requestMarshaled, err := proto.Marshal(&Request{
		Name:        g.Name,
		IdPreimage:  g.IdPreimage.Bytes(),
		KeyPreimage: g.KeyPreimage.Bytes(),
		Members:     g.Members.Serialize(),
		Message:     g.InitMessage,
		Created:     g.Created.UnixNano(),
	})
	if err != nil {
		return nil, NotSent, errors.Errorf(protoMarshalErr, err)
	}

	// Create channel to return the results of each send on
	n := len(g.Members) - 1
	type sendResults struct {
		rounds []id.Round
		err    error
	}
	resultsChan := make(chan sendResults, n)

	// Send request to each member in the group except the leader/sender
	for _, member := range g.Members[1:] {
		go func(member group.Member) {
			rounds, err := m.sendRequest(member.ID, requestMarshaled)
			resultsChan <- sendResults{rounds, err}
		}(member)
	}

	// Block until each send returns
	roundIDs := make(map[id.Round]struct{})
	var errs []string
	for i := 0; i < n; {
		select {
		case results := <-resultsChan:
			for _, rid := range results.rounds {
				roundIDs[rid] = struct{}{}
			}
			if results.err != nil {
				errs = append(errs, results.err.Error())
			}
			i++
		}
	}

	// If all sends returned an error, then return AllFail with a list of errors
	if len(errs) == n {
		return nil, AllFail,
			errors.Errorf(sendRequestAllErr, len(errs), strings.Join(errs, "\n"))
	}

	// If some sends returned an error, then return a list of round IDs for the
	// successful sends and a list of errors for the failed ones
	if len(errs) > 0 {
		return roundIdMap2List(roundIDs), PartialSent,
			errors.Errorf(sendRequestPartialErr, len(errs), n,
				strings.Join(errs, "\n"))
	}

	jww.DEBUG.Printf("Sent group request to %d members in group %q with ID %s.",
		len(g.Members), g.Name, g.ID)

	// If all sends succeeded, return a list of roundIDs
	return roundIdMap2List(roundIDs), AllSent, nil
}

// sendRequest sends the group request to the user via E2E.
func (m Manager) sendRequest(memberID *id.ID, request []byte) ([]id.Round, error) {
	p := e2e.GetDefaultParams()
	p.LastServiceTag = catalog.GroupRq
	p.CMIX.DebugTag = "group.Request"

	rounds, _, _, err := m.e2e.SendE2E(catalog.GroupCreationRequest, m.receptionId, request, p)
	if err != nil {
		return nil, errors.Errorf(sendE2eErr, memberID, err)
	}

	return rounds, nil
}

// roundIdMap2List converts the map of round IDs to a list of round IDs.
func roundIdMap2List(m map[id.Round]struct{}) []id.Round {
	roundIDs := make([]id.Round, 0, len(m))
	for rid := range m {
		roundIDs = append(roundIDs, rid)
	}

	return roundIDs
}
