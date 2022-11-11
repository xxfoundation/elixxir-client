////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/client/partnerships/crust"
	"gitlab.com/elixxir/client/ud"
	crustCrypto "gitlab.com/elixxir/crypto/partnerships/crust"
	"gitlab.com/xx_network/primitives/utils"
	"time"
)

// crustCmd is the subcommand for running backup and restore operations using
// Crust's infrastructure. The operations exist in the partnerships/crust
// package. Specifically crust.RecoverBackup and crust.UploadBackup.
var crustCmd = &cobra.Command{
	Use:   "crust",
	Short: "Backup and restore files using Crust's infrastructure.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmixParams, e2eParams := initParams()
		authCbs := makeAuthCallbacks(
			viper.GetBool(unsafeChannelCreationFlag), e2eParams)
		user := initE2e(cmixParams, e2eParams, authCbs)

		// get identity and save contact to file
		identity := user.GetReceptionIdentity()
		jww.INFO.Printf("[CRUST]User: %s", identity.ID)
		writeContact(identity.GetContact())

		err := user.StartNetworkFollower(50 * time.Millisecond)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		jww.TRACE.Printf("[CRUST] Waiting for connection...")

		// Wait until connected or crash on timeout
		connected := make(chan bool, 10)
		user.GetCmix().AddHealthCallback(
			func(isconnected bool) {
				connected <- isconnected
			})
		waitUntilConnected(connected)

		jww.TRACE.Printf("[CRUST] Connected!")

		// Pull the UD information
		cert, contactFile, address, err := getUdContactInfo(user)
		if err != nil {
			jww.FATAL.Panicf("Failed to load UD contact information from NDF: %+v", err)
		}

		// Initialize the UD manager
		userDiscoveryMgr, err := ud.NewOrLoad(user, user.GetComms(),
			user.NetworkFollowerStatus, "", nil,
			cert, contactFile, address)
		if err != nil {
			jww.FATAL.Panicf("Failed to load or create new UD manager: %+v", err)
		}

		// Retrieve username from UD manager
		username, err := userDiscoveryMgr.GetUsername()
		if err != nil {
			jww.FATAL.Panicf("Failed to retrieve username from UD manager: %+v",
				err)
		}

		// We are assuming here that the user is registered already using the UD
		// subcommand. This should be a set-up step when integration testing
		// Crust-related operations.
		if username == "" {
			jww.FATAL.Panicf("Username from UD manager is empty!")
		}

		jww.INFO.Printf("[CRUST] Retrieved username %v", username)

		// Retrieve the file that either will be uploaded or has been uploaded
		// depending on if triggering the upload or the recover.
		filePath := viper.GetString(crustFile)
		backupFile, err := utils.ReadFile(filePath)
		if err != nil {
			jww.FATAL.Panicf("%v", err)
		}

		// Should not upload and recover in the same process. The recovery
		// process should take place a computationally reasonable amount
		// of time such that the Crust's architecture can process the upload.
		triggerUpload := viper.IsSet(crustUpload)
		triggerRecovery := viper.IsSet(crustRecover)
		if triggerUpload && triggerRecovery {
			jww.FATAL.Panicf("Cannot upload and recover from Crust in the " +
				"same process!")
		}

		// Upload file to Crust's infrastructure
		if triggerUpload {
			// Retrieve private key
			userPrivKey, err := user.GetReceptionIdentity().GetRSAPrivateKey()
			if err != nil {
				jww.FATAL.Panicf("Failed to retrieve private key: %+v", err)
			}

			// Upload file to Crust
			uploadReport, err := crust.UploadBackup(backupFile, userPrivKey,
				userDiscoveryMgr)
			if err != nil {
				jww.FATAL.Panicf("Failed to upload backup to Crust: %+v", err)
			}

			// Marshal upload report (for printing purposes)
			uploadReportJson, err := json.Marshal(uploadReport)
			if err != nil {
				jww.FATAL.Panicf("Failed to marshal upload report: %+v", err)
			}
			jww.INFO.Printf("[CRUST] Upload report: %s", uploadReportJson)
			fmt.Println("Successfully backed up file")
		} else if triggerRecovery {
			// Trigger recovery from Crust
			jww.INFO.Printf("[CRUST] Recovering file!")
			usernameHash := crustCrypto.HashUsername(username)
			recoveredFile, err := crust.RecoverBackup(string(usernameHash))
			if err != nil {
				jww.FATAL.Panicf("Failed to recover backup from Crust: %+v",
					err)
			}

			// Check that recovered file matches originally backed up file.
			if !bytes.Equal(backupFile, recoveredFile) {
				jww.FATAL.Panicf("Recovered file does not match originally "+
					"backed up file!"+
					"\n\tOriginal: %v"+
					"\n\tReceived: %v", backupFile, recoveredFile)
			}

			jww.INFO.Printf("[CRUST] Successfully recovered file")
			fmt.Println("Successfully recovered file")
		}

	},
}

func init() {
	// Crust subcommand options
	crustCmd.Flags().StringP(crustFile, "", "crustBackup.txt",
		"File that will be backed up to Crust's infrastructure. This "+
			"is shared between the upload and recover subcommands. In "+
			"'upload', this will be the the file that will be uploaded to "+
			"Crust's infrastructure. In 'recover' this will check if the "+
			"recovered file matches this passed in file. For the purpose of "+
			"testing recover and upload, this should pass in the same file. "+
			"Defaults to crustBackup.txt if not set.")
	bindFlagHelper(crustFile, crustCmd)

	crustCmd.Flags().Bool(crustRecover, false,
		"Triggers the recovery process for Crust. Setting this and 'upload' "+
			"is undefined behaviour and will cause a panic.")
	bindFlagHelper(crustRecover, crustCmd)

	crustCmd.Flags().Bool(crustUpload, false,
		"Triggers the upload process for Crust. Setting this and 'recover' "+
			"is undefined behaviour and will cause a panic.")
	bindFlagHelper(crustUpload, crustCmd)

	rootCmd.AddCommand(crustCmd)
}
