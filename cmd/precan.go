///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// precan.go handles functions for precan users, which are not usable
// unless you are on a localized test network.
package cmd

import (
	"encoding/binary"
	"gitlab.com/elixxir/client/xxdk/e2eApi"
	"strconv"

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/xx_network/primitives/id"
)

func isPrecanID(id *id.ID) bool {
	// check if precanned
	rBytes := id.Bytes()
	for i := 0; i < 32; i++ {
		if i != 7 && rBytes[i] != 0 {
			return false
		}
	}
	if rBytes[7] != byte(0) && rBytes[7] <= byte(40) {
		return true
	}
	return false
}

// returns a simple numerical id if the user is a precanned user, otherwise
// returns the normal string of the userID
func printIDNice(uid *id.ID) string {

	for index, puid := range precannedIDList {
		if uid.Cmp(puid) {
			return strconv.Itoa(index + 1)
		}
	}

	return uid.String()
}

// build a list of precanned ids to use for comparision for nicer user id output
var precannedIDList = buildPrecannedIDList()

func buildPrecannedIDList() []*id.ID {

	idList := make([]*id.ID, 40)

	for i := 0; i < 40; i++ {
		uid := new(id.ID)
		binary.BigEndian.PutUint64(uid[:], uint64(i+1))
		uid.SetType(id.User)
		idList[i] = uid
	}

	return idList
}

func getPrecanID(recipientID *id.ID) uint {
	return uint(recipientID.Bytes()[7])
}

func addPrecanAuthenticatedChannel(client *e2eApi.E2e, recipientID *id.ID,
	recipient contact.Contact) {
	jww.WARN.Printf("Precanned user id detected: %s", recipientID)
	preUsr, err := client.MakePrecannedAuthenticatedChannel(
		getPrecanID(recipientID))
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	// Sanity check, make sure user id's haven't changed
	preBytes := preUsr.ID.Bytes()
	idBytes := recipientID.Bytes()
	for i := 0; i < len(preBytes); i++ {
		if idBytes[i] != preBytes[i] {
			jww.FATAL.Panicf("no id match: %v %v",
				preBytes, idBytes)
		}
	}
}
