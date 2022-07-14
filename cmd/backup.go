////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package cmd

import (
	"encoding/json"
	"io/fs"
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/client/backup"
	"gitlab.com/elixxir/client/xxdk"
	backupCrypto "gitlab.com/elixxir/crypto/backup"
	"gitlab.com/xx_network/primitives/utils"
)

// loadOrInitBackup will build a new xxdk.E2e from existing storage
// or from a new storage that it will create if none already exists
func loadOrInitBackup(backupPath string, backupPass string, password []byte, storeDir string,
	cmixParams xxdk.CMIXParams, e2eParams xxdk.E2EParams) *xxdk.E2e {
	jww.INFO.Printf("Using Backup sender")

	// create a new client if none exist
	var net *xxdk.Cmix
	var identity xxdk.ReceptionIdentity
	_, err := os.Stat(storeDir)
	if errors.Is(err, fs.ErrNotExist) {
		// Initialize from scratch
		ndfJson, err := ioutil.ReadFile(viper.GetString("ndf"))
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		b, backupFile := loadBackup(backupPath, backupPass)

		// Marshal the backup object in JSON
		backupJson, err := json.Marshal(b)
		if err != nil {
			jww.FATAL.Panicf("Failed to JSON Marshal backup: %+v", err)
		}

		// Write the backup JSON to file
		err = utils.WriteFileDef(viper.GetString("backupJsonOut"), backupJson)
		if err != nil {
			jww.FATAL.Panicf("Failed to write backup to file: %+v", err)
		}

		// Construct client from backup data
		backupIdList, _, err := backup.NewClientFromBackup(string(ndfJson), storeDir,
			password, []byte(backupPass), backupFile)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		backupIdListPath := viper.GetString("backupIdList")
		if backupIdListPath != "" {
			// Marshal backed up ID list to JSON
			backedUpIdListJson, err := json.Marshal(backupIdList)
			if err != nil {
				jww.FATAL.Panicf("Failed to JSON Marshal backed up IDs: %+v", err)
			}

			// Write backed up ID list to file
			err = utils.WriteFileDef(backupIdListPath, backedUpIdListJson)
			if err != nil {
				jww.FATAL.Panicf("Failed to write backed up IDs to file %q: %+v",
					backupIdListPath, err)
			}
		}
	}
	// Initialize from storage
	net, err = xxdk.LoadCmix(storeDir, password, cmixParams)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}

	// Load or initialize xxdk.ReceptionIdentity storage
	identity, err = xxdk.LoadReceptionIdentity(identityStorageKey, net)
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

	messenger, err := xxdk.Login(net, authCbs, identity, e2eParams)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	return messenger
}

func loadBackup(backupPath, backupPass string) (backupCrypto.Backup, []byte) {
	jww.INFO.Printf("Loading backup from path %q", backupPath)
	backupFile, err := utils.ReadFile(backupPath)
	if err != nil {
		jww.FATAL.Panicf("%v", err)
	}

	var b backupCrypto.Backup
	err = b.Decrypt(backupPass, backupFile)
	if err != nil {
		jww.FATAL.Panicf("Failed to decrypt backup: %+v", err)
	}

	return b, backupFile
}
