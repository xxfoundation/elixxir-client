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
	"strings"
)

// Determines what UDB send function to call based on the text in the message
func parseUdbMessage(msg string, client *api.Client) string {
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
		userID, pubKey, err := client.SearchForUser(args[2])
		if err != nil {
			return fmt.Sprintf("UDB search failed: %v", err.Error())
		} else {
			return fmt.Sprintf("UDB search successful. Returned user %v, "+
				"public key %q", *userID, pubKey)
		}
	} else if strings.EqualFold(keyword, "REGISTER") {
		err := client.RegisterForUserDiscovery(args[2])
		if err != nil {
			return fmt.Sprintf("UDB registration failed: %v", err.Error())
		} else {
			return "UDB registration successful."
		}
	} else {
		jww.ERROR.Printf("UDB command not recognized!")
	}
	return ""
}
