////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package io

import (
	"gitlab.com/privategrity/client/globals"
	"gitlab.com/privategrity/comms/client"
	pb "gitlab.com/privategrity/comms/mixmessages"
)

func UpdateUserRegistry(addr string) error {
	contacts, err := client.RequestContactList(addr, &pb.ContactPoll{})
	if err != nil {
		return err
	}
	CheckContacts(contacts)
	return nil
}

// TODO Do we want to remove contacts if they aren't in the retrieved list?
func CheckContacts(contacts *pb.ContactMessage) {
	for _, contact := range contacts.Contacts {
		// upsert nick data into user registry
		user, ok := globals.Users.GetUser(contact.UserID)
		if ok {
			user.Nick = contact.Nick
		} else {
			// the user currently isn't stored in the user registry,
			// so we must make a new one to put in it.
			newUser := globals.User{
				UserID: contact.UserID,
				Nick:   contact.Nick,
			}
			user = &newUser
		}
		globals.Users.UpsertUser(user)
	}
}
