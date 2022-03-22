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
	gs "gitlab.com/elixxir/client/groupChat/groupStore"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/stoppable"
	"gitlab.com/elixxir/crypto/group"
	"time"
)

// Error message.
const (
	sendMessageTypeErr       = "message not of type GroupCreationRequest"
	protoUnmarshalErr        = "failed to unmarshal request: %+v"
	deserializeMembershipErr = "failed to deserialize membership: %+v"
)

// receiveRequest starts the group request reception worker that waits for new
// group requests to arrive.
func (m Manager) receiveRequest(rawMsgs chan message.Receive,
	stop *stoppable.Single) {
	jww.DEBUG.Print("Starting group message request reception worker.")

	for {
		select {
		case <-stop.Quit():
			jww.DEBUG.Print("Stopping group message request reception worker.")
			stop.ToStopped()
			return
		case sendMsg := <-rawMsgs:
			jww.DEBUG.Print("Group message request received message.")

			// Generate the group from the request message
			g, err := m.readRequest(sendMsg)
			if err != nil {
				jww.WARN.Printf("Failed to read message as group request: %+v",
					err)
				continue
			}

			// Call request callback with the new group if it does not already
			// exist
			if _, exists := m.GetGroup(g.ID); !exists {
				jww.DEBUG.Printf("Received group request from sender %s for "+
					"group %s with ID %s.", sendMsg.Sender, g.Name, g.ID)

				go m.requestFunc(g)
			}
		}
	}
}

// readRequest returns the group describes in the group request message. An
// error is returned if the request is of the wrong type or cannot be read.
func (m *Manager) readRequest(msg message.Receive) (gs.Group, error) {
	// Return an error if the message is not of the right type
	if msg.MessageType != message.GroupCreationRequest {
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
	partner, err := m.store.E2e().GetPartner(membership[0].ID)
	if err != nil {
		return gs.Group{}, errors.Errorf(getPrivKeyErr, err)
	}

	// Replace leader's public key with the one from the partnership
	leaderPubKey := membership[0].DhKey.DeepCopy()
	membership[0].DhKey = partner.GetPartnerOriginPublicKey()

	// Generate the DH keys with each group member
	privKey := partner.GetMyOriginPrivateKey()
	grp := m.store.E2e().GetGroup()
	dkl := gs.GenerateDhKeyList(m.gs.GetUser().ID, privKey, membership, grp)

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
