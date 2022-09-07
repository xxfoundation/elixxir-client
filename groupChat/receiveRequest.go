////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package groupChat

import (
	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/catalog"
	"gitlab.com/elixxir/client/e2e/receive"
	gs "gitlab.com/elixxir/client/groupChat/groupStore"
	"gitlab.com/elixxir/crypto/group"
	"time"
)

// Error messages
const (
	sendMessageTypeErr       = "message not of type GroupCreationRequest"
	protoUnmarshalErr        = "failed to unmarshal request: %+v"
	deserializeMembershipErr = "failed to deserialize membership: %+v"
)

// Adheres to receive.Listener interface
type requestListener struct {
	m *manager
}

// Hear waits for new group requests to arrive
func (l *requestListener) Hear(item receive.Message) {
	jww.DEBUG.Print("[GC] Group message request received message.")

	// Generate the group from the request message
	g, err := l.m.readRequest(item)
	if err != nil {
		jww.WARN.Printf(
			"[GC] Failed to read message as group request: %+v", err)
		return
	}

	// Call request callback with the new group if it does not already exist
	if _, exists := l.m.GetGroup(g.ID); !exists {
		jww.INFO.Printf(
			"[GC] Received group request for group %s with ID %s.", g.Name, g.ID)

		l.m.requestFunc(g)
	}
}

// Name returns a name, used for debugging
func (l *requestListener) Name() string {
	return catalog.GroupRq
}

// readRequest returns the group described in the group request message. An
// error is returned if the request is of the wrong type or cannot be read.
func (m *manager) readRequest(msg receive.Message) (gs.Group, error) {
	// Return an error if the message is not of the right type
	if msg.MessageType != catalog.GroupCreationRequest {
		return gs.Group{}, errors.New(sendMessageTypeErr)
	}

	// Unmarshal the request message
	request := &Request{}
	err := proto.Unmarshal(msg.Payload, request)
	if err != nil {
		return gs.Group{}, errors.Errorf(protoUnmarshalErr, err)
	}

	// Deserialize membership list
	membership, err := group.DeserializeMembership(request.GetMembers())
	if err != nil {
		return gs.Group{}, errors.Errorf(deserializeMembershipErr, err)
	}

	// get the relationship with the group leader
	partner, err := m.getE2eHandler().GetPartner(membership[0].ID)
	if err != nil {
		return gs.Group{}, errors.Errorf(getPrivKeyErr, err)
	}

	// Replace leader's public key with the one from the partnership
	leaderPubKey := membership[0].DhKey.DeepCopy()
	membership[0].DhKey = partner.PartnerRootPublicKey()

	// Generate the DH keys with each group member
	privKey := partner.MyRootPrivateKey()
	dkl := gs.GenerateDhKeyList(m.getReceptionIdentity().ID, privKey, membership, m.getE2eGroup())

	// Restore the original public key for the leader so that the membership
	// digest generated later is correct
	membership[0].DhKey = leaderPubKey

	// Copy preimages
	var idPreimage group.IdPreimage
	copy(idPreimage[:], request.GetIdPreimage())
	var keyPreimage group.KeyPreimage
	copy(keyPreimage[:], request.GetKeyPreimage())

	// Create group ID and key
	groupID := group.NewID(idPreimage, membership)
	groupKey := group.NewKey(keyPreimage, membership)

	// Convert created timestamp from nanoseconds to time.Time
	created := time.Unix(0, request.GetCreated())

	// Return the new group
	return gs.NewGroup(request.GetName(), groupID, groupKey, idPreimage,
		keyPreimage, request.GetMessage(), created, membership, dkl), nil
}
