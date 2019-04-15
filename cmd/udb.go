////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package cmd

import (
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/api"
	"gitlab.com/elixxir/primitives/id"
	"strings"
)

func handleSearchResults(user *id.User, pubKey []byte, err error) {
	if err != nil {
		fmt.Printf("UDB search failed: %v\n", err.Error())
	} else if len(pubKey) == 0 {
		fmt.Printf("Public Key returned is empty\n")
	} else {
		fmt.Printf("UDB search successful. Returned user %v\n", *user)
	}
}

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
		client.SearchForUser(args[2], handleSearchResults)
	} else if strings.EqualFold(keyword, "REGISTER") {
		jww.ERROR.Printf("UDB REGISTER not allowed, it is already done during user registration")
	} else {
		jww.ERROR.Printf("UDB command not recognized!")
	}
}
