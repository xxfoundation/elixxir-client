////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package cmd

import (
	"encoding/json"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/client/v4/storage/user"
	"gitlab.com/elixxir/client/v4/xxdk"
	"gitlab.com/xx_network/primitives/utils"
	"io/fs"
	"io/ioutil"
	"os"
)

// loadOrInitProto will build a new xxdk.E2e from existing storage
// or from a new storage that it will create if none already exists
func loadOrInitProto(protoUserPath string, password []byte, storeDir string,
	cmixParams xxdk.CMIXParams, e2eParams xxdk.E2EParams, cbs xxdk.AuthCallbacks) *xxdk.E2e {
	jww.INFO.Printf("Using Proto sender")

	// create a new cMix if none exist
	if _, err := os.Stat(storeDir); errors.Is(err, fs.ErrNotExist) {
		// Initialize from scratch
		ndfJson, err := ioutil.ReadFile(viper.GetString(ndfFlag))
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		protoUserJson, err := utils.ReadFile(protoUserPath)
		if err != nil {
			jww.FATAL.Panicf("%v", err)
		}

		protoUser := &user.Proto{}
		err = json.Unmarshal(protoUserJson, protoUser)
		if err != nil {
			jww.FATAL.Panicf("%v", err)
		}

		err = xxdk.NewProtoCmix_Unsafe(string(ndfJson), storeDir,
			password, protoUser)
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
