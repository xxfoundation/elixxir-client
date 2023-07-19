////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"

	"gitlab.com/elixxir/client/v4/collective"
	"gitlab.com/elixxir/client/v4/collective/remoteSync"
	"gitlab.com/elixxir/client/v4/collective/versioned"
	"gitlab.com/elixxir/client/v4/stoppable"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/ekv"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/netTime"
	"gitlab.com/xx_network/primitives/utils"
)

var remoteSyncCmd = &cobra.Command{
	Use:   "remoteSync",
	Short: "Driver for collective library, uses remove sync server",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		initLog(viper.GetUint(logLevelFlag), viper.GetString(logFlag))
		rngGen := fastRNG.NewStreamGenerator(10, 5, csprng.NewSystemRNG)

		secret := parsePassword(viper.GetString(passwordFlag))
		localPath := viper.GetString(sessionFlag)
		waitTime := time.Duration(viper.GetUint(waitTimeoutFlag)) * time.Second
		username := viper.GetString(remoteUsernameFlag)
		password := viper.GetString(remotePasswordFlag)
		remoteSyncServerAddress := viper.GetString(remoteSyncServerAddressFlag)
		remoteCertPath := viper.GetString(remoteCertPathFlag)
		remoteCert, err := utils.ReadFile(remoteCertPath)
		if err != nil {
			jww.FATAL.Panicf("Failed to read certificate for remote sync "+
				"server from path %s: %+v", remoteCertPath, err)
		}

		// Initialize the sync KV
		fsKV, err := ekv.NewFilestore(localPath, string(secret))
		if err != nil {
			jww.FATAL.Panicf("Failed to make new EKV file store: %+v", err)
		}
		synchronizedPrefixes := []string{"synchronized"}

		params := connect.GetDefaultHostParams()
		params.AuthEnabled = false
		host, err := connect.NewHost(
			&id.DummyUser, remoteSyncServerAddress, remoteCert, params)
		if err != nil {
			jww.FATAL.Panicf("Failed to connect to new host %q: %+v",
				remoteSyncServerAddress, err)
		}
		rng := rngGen.GetStream()
		remote, err := remoteSync.NewRemoteSyncStore(
			username, password, remoteCert, &id.DummyUser, host, rng)
		if err != nil {
			jww.FATAL.Panicf("Failed to log in to remote sync server: %+v", err)
		}
		rng.Close()
		syncKV, err := collective.SynchronizedKV(
			"", secret, remote, fsKV, synchronizedPrefixes, rngGen)
		if err != nil {
			jww.FATAL.Panicf("Failed to create new synchronized KV: %+v", err)
		}

		key := viper.GetString(remoteSyncKey)
		val := viper.GetString(remoteSyncVal)

		// Set up the prefix by splitting on /
		parts := strings.Split(key, "/")
		jww.INFO.Printf("Parts: %s", parts)
		var kv versioned.KV
		kv = syncKV
		jww.INFO.Printf("Prefixing: %s", key)
		for _, part := range parts[:len(parts)-1] {
			jww.INFO.Printf("Part: %s", part)
			kv, err = kv.Prefix(part)
			if err != nil {
				jww.FATAL.Printf("Prefix failure for part %s: %+v", part, err)
			}
			jww.INFO.Printf("Prefix Set: %s", kv.GetPrefix())
			fmt.Printf("Prefix Set: %s\n", kv.GetPrefix())
		}
		key = parts[len(parts)-1]

		// Listen on the key
		waitCh := make(chan struct {
			old, new *versioned.Object
			op       versioned.KeyOperation
		})
		cb := func(old, new *versioned.Object, op versioned.KeyOperation) {
			waitCh <- struct {
				old, new *versioned.Object
				op       versioned.KeyOperation
			}{old: old, new: new, op: op}
		}
		err = kv.ListenOnRemoteKey(parts[len(parts)-1], 0, cb, false)
		if err != nil {
			jww.FATAL.Printf("Failed to listen on remote key %q: %+v",
				parts[len(parts)-1], err)
		}

		// Begin synchronization
		stopSync, err := syncKV.StartProcesses()
		if err != nil {
			jww.FATAL.Panicf("Failed to start KV synchronization: %+v", err)
		}

		// Set the actual key if a value exists
		if val != "" {
			obj := &versioned.Object{
				Timestamp: netTime.Now(),
				Version:   0,
				Data:      []byte(val),
			}
			err = kv.Set(key, obj)
			if err != nil {
				jww.FATAL.Panicf("Failed to set object to key %q: %+v", key, err)
			} else {
				jww.INFO.Printf("Set object to key %q: %s", key, voStr(obj))
				fmt.Printf("Set object to key %q: %s\n", key, voStrNoTS(obj))
			}
			time.Sleep(6 * time.Second)
		}
		// Wait for updates or timeout
		synced := syncKV.WaitForRemote(waitTime)
		if !synced {
			jww.ERROR.Printf("syncKV timed out waiting for remote")
		}

		if val == "" {
			jww.INFO.Printf("No value, waiting for update")
			select {
			case <-time.After(waitTime):
				jww.ERROR.Printf(
					"Timed out after %s waiting for key update.", waitTime)
			case update := <-waitCh:
				jww.INFO.Printf("Update received for %q (%s): %s -> %s",
					key, update.op, voStr(update.old), voStr(update.new))
				fmt.Printf("Update received for %q (%s): %s -> %s\n",
					key, update.op, voStrNoTS(update.old), voStrNoTS(update.new))
			}
		}

		endVal, err := kv.Get(key, 0)
		if err != nil {
			jww.ERROR.Printf("Get end value error: %+v", err)
		} else {
			jww.INFO.Printf("End Value for %q: %s", key, voStr(endVal))
			fmt.Printf("End Value for %q: %s\n", key, voStrNoTS(endVal))
		}
		err = stopSync.Close()
		if err != nil {
			jww.ERROR.Printf("Failed to close sync stoppable: %+v", err)
		}
		err = stoppable.WaitForStopped(stopSync, 2*time.Second)
		if err != nil {
			jww.FATAL.Panicf("Timed out waiting for sync stop: %+v", err)
		}
	},
}

const (
	remoteSyncKey = "remoteKey"
	remoteSyncVal = "remoteValue"

	remoteSyncServerAddressFlag = "remoteSyncServerAddress"
	remoteCertPathFlag          = "remoteCertPath"
	remoteUsernameFlag          = "remoteUsername"
	remotePasswordFlag          = "remotePassword"
)

func init() {
	flags := remoteSyncCmd.Flags()

	flags.String(remoteSyncKey, "DefaultKey2", "Key to set or get")
	bindFlagHelper(remoteSyncKey, remoteSyncCmd)

	flags.String(remoteSyncVal, "", "Set to value, otherwise get")
	bindFlagHelper(remoteSyncVal, remoteSyncCmd)

	flags.String(remoteSyncServerAddressFlag, "0.0.0.0:22841",
		"Address to remote sync server.")
	bindFlagHelper(remoteSyncServerAddressFlag, remoteSyncCmd)
	flags.String(remoteCertPathFlag, "",
		"PEM encoded certificate for remote sync server.")
	bindFlagHelper(remoteCertPathFlag, remoteSyncCmd)

	flags.String(remoteUsernameFlag, "", "Username for remote sync server.")
	bindFlagHelper(remoteUsernameFlag, remoteSyncCmd)
	flags.String(remotePasswordFlag, "", "Password for remote sync server.")
	bindFlagHelper(remotePasswordFlag, remoteSyncCmd)

	rootCmd.AddCommand(remoteSyncCmd)
}

// voStr returns a printable string of the versioned.Object.
func voStr(vo *versioned.Object) string {
	if vo == nil {
		return "nil"
	}

	fields := []string{
		"Version:" + strconv.FormatUint(vo.Version, 10),
		"Timestamp:" + vo.Timestamp.String(),
		"Data:" + fmt.Sprintf("%q", vo.Data),
	}

	return "{" + strings.Join(fields, " ") + "}"
}

// voStrNoTS returns a printable string of the versioned.Object without the
// timestamp
func voStrNoTS(vo *versioned.Object) string {
	if vo == nil {
		return "nil"
	}

	fields := []string{
		"Version:" + strconv.FormatUint(vo.Version, 10),
		"Data:" + fmt.Sprintf("%q", vo.Data),
	}

	return "{" + strings.Join(fields, " ") + "}"
}
