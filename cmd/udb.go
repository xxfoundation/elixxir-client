////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package cmd

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/bindings"
	"strings"
	"fmt"
	"gitlab.com/elixxir/crypto/cyclic"
)

// Determines what UDB send function to call based on the text in the message
func parseUdbMessage(msg string) string {
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
		result, err := bindings.SearchForUser(args[2])
		if err != nil {
			return fmt.Sprintf("UDB search failed: %v", err.Error())
		} else {
			userIdText := cyclic.NewIntFromBytes(result.ResultID).Text(10)
			return fmt.Sprintf("UDB search successful. Returned user %v, "+
				"public key %q", userIdText, result.PublicKey)
		}
	} else if strings.EqualFold(keyword, "REGISTER") {
		err := bindings.RegisterForUserDiscovery(args[2])
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
