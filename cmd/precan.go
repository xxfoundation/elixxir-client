////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

// precan.go handles functions for precan users, which are not usable
// unless you are on a localized test network.
package cmd

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/client/v4/xxdk"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/xx_network/primitives/id"
	"io/fs"
	"io/ioutil"
	"os"
)

// loadOrInitPrecan will build a new xxdk.E2e from existing storage
// or from a new storage that it will create if none already exists
func loadOrInitPrecan(precanId uint, password []byte, storeDir string,
	cmixParams xxdk.CMIXParams, e2eParams xxdk.E2EParams, cbs xxdk.AuthCallbacks) *xxdk.E2e {
	jww.INFO.Printf("Using Precanned sender")

	// create a new cMix if none exist
	if _, err := os.Stat(storeDir); errors.Is(err, fs.ErrNotExist) {
		// Initialize from scratch
		ndfJson, err := ioutil.ReadFile(viper.GetString(ndfFlag))
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		err = xxdk.NewPrecannedCmix(precanId, string(ndfJson), storeDir, password)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}
	}
	// Initialize from storage
	net, err := xxdk.LoadCmix(storeDir, password, cmixParams)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}

	// Load or initialize xxdk.ReceptionIdentity storage
	identity, err := xxdk.LoadReceptionIdentity(identityStorageKey, net)
	if err != nil {
		identity, err = xxdk.MakeLegacyReceptionIdentity(net)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		err = xxdk.StoreReceptionIdentity(identityStorageKey, identity, net)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}
	}

	user, err := xxdk.Login(net, cbs, identity, e2eParams)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	return user
}

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

func getPrecanID(recipientID *id.ID) uint {
	return uint(recipientID.Bytes()[7])
}

func addPrecanAuthenticatedChannel(user *xxdk.E2e, recipientID *id.ID,
	recipient contact.Contact) {
	jww.WARN.Printf("Precanned user id detected: %s", recipientID)
	preUsr, err := user.MakePrecannedAuthenticatedChannel(
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
