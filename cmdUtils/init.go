package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/client/backup"
	"gitlab.com/elixxir/client/storage/user"
	"gitlab.com/elixxir/client/xxdk"
	backupCrypto "gitlab.com/elixxir/crypto/backup"
	"gitlab.com/xx_network/primitives/utils"
	"io/fs"
	"io/ioutil"
	"os"
)

// InitE2e returns a fully-formed xxdk.E2e object
func InitE2e(cmixParams xxdk.CMIXParams, e2eParams xxdk.E2EParams,
	callbacks xxdk.AuthCallbacks) *xxdk.E2e {

	// Intake parameters for client initialization
	precanId := viper.GetUint(sendIdFlag)
	protoUserPath := viper.GetString(protoUserPathFlag)
	userIdPrefix := viper.GetString(userIdPrefixFlag)
	backupPath := viper.GetString(backupInFlag)
	backupPass := viper.GetString(backupPassFlag)
	storePassword := ParsePassword(viper.GetString(PasswordFlag))
	storeDir := viper.GetString(SessionFlag)
	regCode := viper.GetString(RegCodeFlag)
	forceLegacy := viper.GetBool(ForceLegacyFlag)
	jww.DEBUG.Printf("sessionDir: %v", storeDir)

	// Initialize the client of the proper type
	var messenger *xxdk.E2e
	if precanId != 0 {
		messenger = loadOrInitPrecan(precanId, storePassword, storeDir, cmixParams, e2eParams, callbacks)
	} else if protoUserPath != "" {
		messenger = loadOrInitProto(protoUserPath, storePassword, storeDir, cmixParams, e2eParams, callbacks)
	} else if userIdPrefix != "" {
		messenger = loadOrInitVanity(storePassword, storeDir, regCode, userIdPrefix, cmixParams, e2eParams, callbacks)
	} else if backupPath != "" {
		messenger = loadOrInitBackup(backupPath, backupPass, storePassword, storeDir, cmixParams, e2eParams, callbacks)
	} else {
		messenger = loadOrInitMessenger(forceLegacy, storePassword, storeDir, regCode, cmixParams, e2eParams, callbacks)
	}

	// Handle protoUser output
	if protoUser := viper.GetString(protoUserOutFlag); protoUser != "" {
		jsonBytes, err := messenger.ConstructProtoUserFile()
		if err != nil {
			jww.FATAL.Panicf("cannot construct proto user file: %v",
				err)
		}

		err = utils.WriteFileDef(protoUser, jsonBytes)
		if err != nil {
			jww.FATAL.Panicf("cannot write proto user to file: %v",
				err)
		}
	}

	// Handle backup output
	if backupOut := viper.GetString("backupOutFlag"); backupOut != "" {
		if !forceLegacy {
			jww.FATAL.Panicf("Unable to make backup for non-legacy sender!")
		}
		updateBackupCb := func(encryptedBackup []byte) {
			jww.INFO.Printf("Backup update received, size %d",
				len(encryptedBackup))
			fmt.Println("Backup update received.")
			err := utils.WriteFileDef(backupOut, encryptedBackup)
			if err != nil {
				jww.FATAL.Panicf("cannot write backup: %+v",
					err)
			}

			backupJsonPath := viper.GetString(backupJsonOutFlag)

			if backupJsonPath != "" {
				var b backupCrypto.Backup
				err = b.Decrypt(backupPass, encryptedBackup)
				if err != nil {
					jww.ERROR.Printf("cannot decrypt backup: %+v", err)
				}

				backupJson, err := json.Marshal(b)
				if err != nil {
					jww.ERROR.Printf("Failed to JSON unmarshal backup: %+v", err)
				}

				err = utils.WriteFileDef(backupJsonPath, backupJson)
				if err != nil {
					jww.FATAL.Panicf("Failed to write backup to file: %+v", err)
				}
			}
		}
		_, err := backup.InitializeBackup(backupPass, updateBackupCb,
			messenger.GetBackupContainer(), messenger.GetE2E(), messenger.GetStorage(),
			nil, messenger.GetStorage().GetKV(), messenger.GetRng())
		if err != nil {
			jww.FATAL.Panicf("Failed to initialize backup with key %q: %+v",
				backupPass, err)
		}
	}

	return messenger
}

// LoadOrInitCmix will build a new xxdk.Cmix from existing storage
// or from a new storage that it will create if none already exists
func LoadOrInitCmix(password []byte, storeDir, regCode string,
	cmixParams xxdk.CMIXParams) *xxdk.Cmix {
	// create a new client if none exist
	if _, err := os.Stat(storeDir); errors.Is(err, fs.ErrNotExist) {
		// Initialize from scratch
		ndfJson, err := ioutil.ReadFile(viper.GetString(NdfFlag))
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		err = xxdk.NewCmix(string(ndfJson), storeDir, password, regCode)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}
	}

	// Initialize from storage
	net, err := xxdk.LoadCmix(storeDir, password, cmixParams)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}

	return net
}

// LoadOrInitReceptionIdentity will build a new xxdk.ReceptionIdentity from existing storage
// or from a new storage that it will create if none already exists
func LoadOrInitReceptionIdentity(forceLegacy bool, net *xxdk.Cmix) xxdk.ReceptionIdentity {
	// Load or initialize xxdk.ReceptionIdentity storage
	identity, err := xxdk.LoadReceptionIdentity(IdentityStorageKey, net)
	if err != nil {
		if forceLegacy {
			jww.INFO.Printf("Forcing legacy sender")
			identity, err = xxdk.MakeLegacyReceptionIdentity(net)
		} else {
			identity, err = xxdk.MakeReceptionIdentity(net)
		}
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		err = xxdk.StoreReceptionIdentity(IdentityStorageKey, identity, net)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}
	}
	return identity
}

// loadOrInitPrecan will build a new xxdk.E2e from existing storage
// or from a new storage that it will create if none already exists
func loadOrInitPrecan(precanId uint, password []byte, storeDir string,
	cmixParams xxdk.CMIXParams, e2eParams xxdk.E2EParams, cbs xxdk.AuthCallbacks) *xxdk.E2e {
	jww.INFO.Printf("Using Precanned sender")

	// create a new client if none exist
	if _, err := os.Stat(storeDir); errors.Is(err, fs.ErrNotExist) {
		// Initialize from scratch
		ndfJson, err := ioutil.ReadFile(viper.GetString(NdfFlag))
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		err = xxdk.NewPrecannedClient(precanId, string(ndfJson), storeDir, password)
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
	identity, err := xxdk.LoadReceptionIdentity(IdentityStorageKey, net)
	if err != nil {
		identity, err = xxdk.MakeLegacyReceptionIdentity(net)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		err = xxdk.StoreReceptionIdentity(IdentityStorageKey, identity, net)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}
	}

	messenger, err := xxdk.Login(net, cbs, identity, e2eParams)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	return messenger
}

// loadOrInitProto will build a new xxdk.E2e from existing storage
// or from a new storage that it will create if none already exists
func loadOrInitProto(protoUserPath string, password []byte, storeDir string,
	cmixParams xxdk.CMIXParams, e2eParams xxdk.E2EParams, cbs xxdk.AuthCallbacks) *xxdk.E2e {
	jww.INFO.Printf("Using Proto sender")

	// create a new client if none exist
	if _, err := os.Stat(storeDir); errors.Is(err, fs.ErrNotExist) {
		// Initialize from scratch
		ndfJson, err := ioutil.ReadFile(viper.GetString(NdfFlag))
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

		err = xxdk.NewProtoClient_Unsafe(string(ndfJson), storeDir,
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
	identity, err := xxdk.LoadReceptionIdentity(IdentityStorageKey, net)
	if err != nil {
		identity, err = xxdk.MakeLegacyReceptionIdentity(net)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		err = xxdk.StoreReceptionIdentity(IdentityStorageKey, identity, net)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}
	}

	messenger, err := xxdk.Login(net, cbs, identity, e2eParams)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	return messenger
}

// loadOrInitVanity will build a new xxdk.E2e from existing storage
// or from a new storage that it will create if none already exists
func loadOrInitVanity(password []byte, storeDir, regCode, userIdPrefix string,
	cmixParams xxdk.CMIXParams, e2eParams xxdk.E2EParams, cbs xxdk.AuthCallbacks) *xxdk.E2e {
	jww.INFO.Printf("Using Vanity sender")

	// create a new client if none exist
	if _, err := os.Stat(storeDir); errors.Is(err, fs.ErrNotExist) {
		// Initialize from scratch
		ndfJson, err := ioutil.ReadFile(viper.GetString(NdfFlag))
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		err = xxdk.NewVanityClient(string(ndfJson), storeDir,
			password, regCode, userIdPrefix)
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
	identity, err := xxdk.LoadReceptionIdentity(IdentityStorageKey, net)
	if err != nil {
		identity, err = xxdk.MakeLegacyReceptionIdentity(net)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		err = xxdk.StoreReceptionIdentity(IdentityStorageKey, identity, net)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}
	}

	messenger, err := xxdk.Login(net, cbs, identity, e2eParams)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	return messenger
}

// loadOrInitBackup will build a new xxdk.E2e from existing storage
// or from a new storage that it will create if none already exists
func loadOrInitBackup(backupPath string, backupPass string, password []byte, storeDir string,
	cmixParams xxdk.CMIXParams, e2eParams xxdk.E2EParams, cbs xxdk.AuthCallbacks) *xxdk.E2e {
	jww.INFO.Printf("Using Backup sender")

	// create a new client if none exist
	if _, err := os.Stat(storeDir); errors.Is(err, fs.ErrNotExist) {
		// Initialize from scratch
		ndfJson, err := ioutil.ReadFile(viper.GetString(NdfFlag))
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
		err = utils.WriteFileDef(viper.GetString(backupJsonOutFlag), backupJson)
		if err != nil {
			jww.FATAL.Panicf("Failed to write backup to file: %+v", err)
		}

		// Construct client from backup data
		backupIdList, _, err := backup.NewClientFromBackup(string(ndfJson), storeDir,
			password, []byte(backupPass), backupFile)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		backupIdListPath := viper.GetString(backupIdListFlag)
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
	net, err := xxdk.LoadCmix(storeDir, password, cmixParams)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}

	// Load or initialize xxdk.ReceptionIdentity storage
	identity, err := xxdk.LoadReceptionIdentity(IdentityStorageKey, net)
	if err != nil {
		identity, err = xxdk.MakeLegacyReceptionIdentity(net)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		err = xxdk.StoreReceptionIdentity(IdentityStorageKey, identity, net)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}
	}

	messenger, err := xxdk.Login(net, cbs, identity, e2eParams)
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

// LoadOrInitMessenger will build a new xxdk.E2e from existing storage
// or from a new storage that it will create if none already exists
func loadOrInitMessenger(forceLegacy bool, password []byte, storeDir, regCode string,
	cmixParams xxdk.CMIXParams, e2eParams xxdk.E2EParams, cbs xxdk.AuthCallbacks) *xxdk.E2e {
	jww.INFO.Printf("Using normal sender")

	net := LoadOrInitCmix(password, storeDir, regCode, cmixParams)
	identity := LoadOrInitReceptionIdentity(forceLegacy, net)

	messenger, err := xxdk.Login(net, cbs, identity, e2eParams)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	return messenger
}
