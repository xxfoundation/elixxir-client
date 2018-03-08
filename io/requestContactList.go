////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
jww "github.com/spf13/jwalterweatherman"
"gitlab.com/privategrity/comms/mixclient"
pb "gitlab.com/privategrity/comms/mixmessages"
)

func GetContactList(addr string) (uids []uint64, nicks []string) {
	contacts, err := mixclient.RequestContactList(addr, &pb.ContactPoll{})

	if err != nil {
		jww.FATAL.Panicf("Couldn't get contact list from server: %s", err.Error())
	}

	uids = make([]uint64, len(contacts.Contacts))
	nicks = make([]string, len(contacts.Contacts))
	for i, contact := range (contacts.Contacts) {
		uids[i] = contact.UserID
		nicks[i] = contact.Nick
	}

	return uids, nicks
}
