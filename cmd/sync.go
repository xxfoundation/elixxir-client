////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

// The group subcommand allows creation and sending messages to groups

package cmd

import (
	"encoding/json"
	"strings"
	"time"

	"gitlab.com/elixxir/client/v4/collective"
	"gitlab.com/elixxir/client/v4/stoppable"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/crypto/csprng"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
)

// Sync Specific command line options
const (
	syncKey        = "key"
	syncVal        = "value"
	syncRemotePath = "remote"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Driver for collective library, uses local fs for remote",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		initLog(viper.GetUint(logLevelFlag), viper.GetString(logFlag))
		rngGen := fastRNG.NewStreamGenerator(10, 5, csprng.NewSystemRNG)

		secret := parsePassword(viper.GetString(passwordFlag))
		remotePath := viper.GetString(syncRemotePath)
		localPath := viper.GetString(sessionFlag)
		waitTime := time.Duration(time.Duration(
			viper.GetUint(waitTimeoutFlag)) * time.Second)
		// Initialize the sync kv
		fskv, err := ekv.NewFilestore(localPath, string(secret))
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}
		synchronizedPrefixes := []string{"synchronized"}
		remote := collective.NewFileSystemRemoteStorage(remotePath)
		synckv, err := collective.SynchronizedKV(remotePath, secret,
			remote, fskv, synchronizedPrefixes, rngGen)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		key := viper.GetString(syncKey)
		val := viper.GetString(syncVal)

		// Set up the prefix by splitting on /
		parts := strings.Split(key, "/")
		jww.INFO.Printf("Parts: %v", parts)
		var kv versioned.KV
		kv = synckv
		jww.INFO.Printf("Prefixing: %s", key)
		for i := 0; i < len(parts)-1; i++ {
			part := parts[i]
			jww.INFO.Printf("Part: %s", part)
			kv, err = kv.Prefix(part)
			if err != nil {
				jww.FATAL.Printf("prefix failure %s: %+v",
					parts[i], err)
			}
			jww.INFO.Printf("Prefix Set: %s", kv.GetPrefix())
		}
		key = parts[len(parts)-1]

		// Listen on the key
		waitCh := make(chan bool)
		cb := func(key string, old, new *versioned.Object,
			op versioned.KeyOperation) {
			oldJSON, _ := json.Marshal(old)
			newJSON, _ := json.Marshal(new)
			jww.INFO.Printf("Update received for %s (%v): %s -> %s",
				key, op, oldJSON, newJSON)
			waitCh <- false
		}
		kv.ListenOnRemoteKey(parts[len(parts)-1], 0, cb)

		// Begin synchronization
		stopSync, err := synckv.StartProcesses()
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		// Set the actual key if a value exists
		if val != "" {
			kv.Set(key, &versioned.Object{
				Timestamp: time.Now(),
				Version:   0,
				Data:      []byte(val),
			})
			time.Sleep(6 * time.Second)
		}
		// Wait for updates or timeout
		synched := synckv.WaitForRemote(waitTime)
		if !synched {
			jww.ERROR.Printf("synckv timed out waiting for remote")
		}

		if val == "" {
			jww.INFO.Printf("no value, waiting for update")
			select {
			case <-time.After(waitTime):
				jww.ERROR.Printf("did not get key update")
			case <-waitCh:
				jww.INFO.Printf("got key update")
			}
		}

		endVal, err := kv.Get(key, 0)
		if err != nil {
			jww.INFO.Printf("end value error: %v", err)
		} else {
			evJSON, _ := json.Marshal(endVal)
			jww.INFO.Printf("End Value for %s: %s",
				key, evJSON)
			jww.INFO.Printf("Data Decoded: %s",
				string(endVal.Data))
		}
		stopSync.Close()
		err = stoppable.WaitForStopped(stopSync, 2*time.Second)
		if err != nil {
			jww.FATAL.Panicf("timed out waiting for sync stop: %+v",
				err)
		}
	},
}

func init() {
	flags := syncCmd.Flags()
	flags.StringP(syncRemotePath, "r", "RemoteStore",
		"Synthetic remote storage path, directory on disk")
	viper.BindPFlag(syncRemotePath, flags.Lookup(syncRemotePath))

	flags.StringP(syncKey, "k", "DefaultKey", "Key to set or get")
	viper.BindPFlag(syncKey, flags.Lookup(syncKey))

	flags.StringP(syncVal, "", "", "Set to value, otherwise get")
	viper.BindPFlag(syncVal, flags.Lookup(syncVal))

	rootCmd.AddCommand(syncCmd)
}
