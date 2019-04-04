////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package cmd

import (
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/bindings"
	"gitlab.com/elixxir/crypto/large"
	"strings"
)

func handleSearchResults(result bindings.SearchResult, err error) {
	if err != nil {
		fmt.Printf("UDB search failed: %v\n", err.Error())
	} else {
		userIdText := large.NewIntFromBytes(result.ResultID).Text(10)
		fmt.Printf("UDB search successful. Returned user %v, "+
			"public key %q\n", userIdText, result.PublicKey)
	}
}

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
		bindings.SearchForUser(args[2], handleSearchResults)
	} else if strings.EqualFold(keyword, "REGISTER") {
		jww.ERROR.Printf("UDB REGISTER not allowed, it is already done during user registration")
	} else {
		jww.ERROR.Printf("UDB command not recognized!")
	}
}
