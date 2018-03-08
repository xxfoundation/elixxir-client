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
	"gitlab.com/privategrity/client/globals"
)

func UpdateUserRegistry(addr string) {
	contacts, err := mixclient.RequestContactList(addr, &pb.ContactPoll{})

	if err != nil {
		jww.FATAL.Panicf("Couldn't get contact list from server: %s", err.Error())
	}

	for _, contact := range (contacts.Contacts) {
		// upsert nick data into user registry
		user, ok := globals.Users.GetUser(contact.UserID)
		if ok {
			user.Nick = contact.Nick
		} else {
			// the user currently isn't stored in the user registry,
			// so we must make a new one to put in it.
			newUser := globals.User(*contact)
			user = &newUser
		}
		// TODO implement this
		globals.Users.UpsertUser(user)
	}
}
