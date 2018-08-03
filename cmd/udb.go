////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package cmd

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/privategrity/client/bindings"
	"strings"
)

// Determines what UDB send function to call based on the text in the message
func parseUdbMessage(msg string) {
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
		bindings.SearchForUser(args[2])
	} else if strings.EqualFold(keyword, "REGISTER") {
		bindings.RegisterForUserDiscovery(args[2])
	} else {
		jww.ERROR.Printf("UDB command not recognized!")
	}
}
