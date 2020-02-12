////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package cmd

import (
	"gitlab.com/elixxir/client/api"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/primitives/id"
	"strings"
	"time"
)

type callbackSearch struct{}

func (cs callbackSearch) Callback(userID, pubKey []byte, err error) {
	if err != nil {
		globals.Log.INFO.Printf("UDB search failed: %v\n", err.Error())
	} else if len(pubKey) == 0 {
		globals.Log.INFO.Printf("Public Key returned is empty\n")
	} else {
		globals.Log.INFO.Printf("UDB search successful. Returned user %v\n",
			*id.NewUserFromBytes(userID))
	}
}

var searchCallback = callbackSearch{}

// Determines what UDB send function to call based on the text in the message
func parseUdbMessage(msg string, client *api.Client) {
	// Split the message on spaces
	args := strings.Fields(msg)
	if len(args) < 3 {
		globals.Log.ERROR.Printf("UDB command must have at least three arguments!")
	}
	// The first arg is the command
	// the second is the valueType
	// the third is the value
	keyword := args[0]
	// Case-insensitive match the keyword to a command
	if strings.EqualFold(keyword, "SEARCH") {
		client.SearchForUser(args[2], searchCallback, 2*time.Minute)
	} else if strings.EqualFold(keyword, "REGISTER") {
		globals.Log.ERROR.Printf("UDB REGISTER not allowed, it is already done during user registration")
	} else {
		globals.Log.ERROR.Printf("UDB command not recognized!")
	}
}
