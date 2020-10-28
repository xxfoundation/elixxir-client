////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package cmd

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/api"
	"gitlab.com/xx_network/primitives/id"
	"strings"
	//"time"
)

type callbackSearch struct{}

func (cs callbackSearch) Callback(userID, pubKey []byte, err error) {
	if err != nil {
		jww.INFO.Printf("UDB search failed: %v\n", err.Error())
	} else if len(pubKey) == 0 {
		jww.INFO.Printf("Public Key returned is empty\n")
	} else {
		userID, err := id.Unmarshal(userID)
		if err != nil {
			jww.ERROR.Printf("Malformed user ID from successful UDB search: %v", err)
		}
		jww.INFO.Printf("UDB search successful. Returned user %v\n",
			userID)
	}
}

var searchCallback = callbackSearch{}

// Determines what UDB send function to call based on the text in the message
func parseUdbMessage(msg string, client *api.Client) {
	// Split the message on spaces
	args := strings.Fields(msg)
	if len(args) < 3 {
		jww.ERROR.Printf("UDB command must have at least three arguments!")
	}
	// The first arg is the command
	// the second is the valueType
	// the third is the value
	keyword := args[0]
	// Case-insensitive match the keyword to a command
	if strings.EqualFold(keyword, "SEARCH") {
		//client.SearchForUser(args[2], searchCallback, 2*time.Minute)
	} else if strings.EqualFold(keyword, "REGISTER") {
		jww.ERROR.Printf("UDB REGISTER not allowed, it is already done during user registration")
	} else {
		jww.ERROR.Printf("UDB command not recognized!")
	}
}
