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

func GetContactList(addr string) []*pb.Contact {
	contacts, err := mixclient.RequestContactList(addr, &pb.ContactPoll{})

	if err != nil {
		jww.FATAL.Panicf("Couldn't get contact list from server: %s", err.Error())
	}

	return contacts.Contacts
}
