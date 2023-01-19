////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

// Package cmd initializes the CLI and config parsers as well as the logger.
package cmd

import (
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/xxdk"
	"gitlab.com/xx_network/primitives/id"
	"time"

	"gitlab.com/elixxir/client/single"
	"gitlab.com/elixxir/client/ud"
	"gitlab.com/elixxir/client/xxmutils"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/primitives/utils"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/crypto/contact"
)

// udCmd is the user discovery subcommand, which allows for user lookup,
// registration, and search. This basically runs a client for these functions
// with the UD module enabled. Normally, clients do not need it, so it is not
// loaded for the rest of the commands.
var udCmd = &cobra.Command{
	Use:   "ud",
	Short: "Register for and search users using the xx network user discovery service.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmixParams, e2eParams := initParams()
		authCbs := makeAuthCallbacks(
			viper.GetBool(unsafeChannelCreationFlag), e2eParams)
		user := initE2e(cmixParams, e2eParams, authCbs)

		// get identity and save contact to file
		identity := user.GetReceptionIdentity()
		jww.INFO.Printf("[UD]User: %s", identity.ID)
		writeContact(identity.GetContact())

		err := user.StartNetworkFollower(50 * time.Millisecond)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		jww.TRACE.Printf("[UD] Waiting for connection...")

		// Wait until connected or crash on timeout
		connected := make(chan bool, 10)
		user.GetCmix().AddHealthCallback(
			func(isconnected bool) {
				connected <- isconnected
			})
		waitUntilConnected(connected)

		jww.TRACE.Printf("[UD] Connected!")

		cert, contactFile, address, err := getUdContactInfo(user)
		if err != nil {
			jww.FATAL.Panicf("Failed to load UD contact information from NDF: %+v", err)
		}

		// Make user discovery manager
		userToRegister := viper.GetString(udRegisterFlag)
		jww.TRACE.Printf("[UD] Registering identity %v...", userToRegister)
		userDiscoveryMgr, err := ud.NewOrLoad(user, user.GetComms(),
			user.NetworkFollowerStatus, userToRegister, nil,
			cert, contactFile, address)
		if err != nil {
			jww.FATAL.Panicf("Failed to load or create new UD manager: %+v", err)
		}
		jww.INFO.Printf("[UD] Registered user %v", userToRegister)

		var newFacts fact.FactList
		phone := viper.GetString(udAddPhoneFlag)
		if phone != "" {
			f, err := fact.NewFact(fact.Phone, phone)
			if err != nil {
				jww.FATAL.Panicf("Failed to create new fact: %+v", err)
			}
			newFacts = append(newFacts, f)
		}

		email := viper.GetString(udAddEmailFlag)
		if email != "" {
			f, err := fact.NewFact(fact.Email, email)
			if err != nil {
				jww.FATAL.Panicf("Failed to create new fact: %+v", err)
			}
			newFacts = append(newFacts, f)
		}

		for i := 0; i < len(newFacts); i++ {
			jww.INFO.Printf("[UD] Registering Fact: %v",
				newFacts[i])
			r, err := userDiscoveryMgr.SendRegisterFact(newFacts[i])
			if err != nil {
				fmt.Printf("Failed to register fact: %s\n",
					newFacts[i])
				jww.FATAL.Panicf("[UD] Failed to send register fact: %+v", err)
			}
			// TODO Store the code?
			jww.INFO.Printf("[UD] Fact Add Response: %+v", r)
		}

		confirmID := viper.GetString(udConfirmFlag)
		if confirmID != "" {
			jww.INFO.Printf("[UD] Confirming fact: %v", confirmID)
			err = userDiscoveryMgr.ConfirmFact(confirmID, confirmID)
			if err != nil {
				fmt.Printf("Couldn't confirm fact: %s\n",
					err.Error())
				jww.FATAL.Panicf("%+v", err)
			}

			jww.INFO.Printf("[UD] Confirmed %v", confirmID)
		}

		udContact := userDiscoveryMgr.GetContact()

		// Handle lookup (verification) process
		// Note: Cryptographic verification occurs above the bindings layer
		lookupIDStr := viper.GetString(udLookupFlag)
		if lookupIDStr != "" {
			lookupID := parseRecipient(lookupIDStr)
			jww.INFO.Printf("[UD] Looking up %v", lookupID)

			cb := func(newContact contact.Contact, err error) {
				if err != nil {
					jww.FATAL.Panicf("UserDiscovery Lookup error: %+v", err)
				}
				printContact(newContact)
			}

			_, _, err = ud.Lookup(user,
				udContact, cb, lookupID, single.GetDefaultRequestParams())
			if err != nil {
				jww.WARN.Printf("Failed UD lookup: %+v", err)
			}

			time.Sleep(31 * time.Second)
		}

		if viper.IsSet(udBatchAddFlag) {
			idListFile, err := utils.ReadFile(viper.GetString(udBatchAddFlag))
			if err != nil {
				fmt.Printf("BATCHADD: Couldn't read file: %s\n",
					err.Error())
				jww.FATAL.Panicf("BATCHADD: Couldn't read file: %+v", err)
			}
			jww.INFO.Printf("[UD] BATCHADD: Running")
			restored, _, _, err := xxmutils.RestoreContactsFromBackup(
				idListFile, user, userDiscoveryMgr, nil)
			if err != nil {
				jww.FATAL.Panicf("%+v", err)
			}
			for i := 0; i < len(restored); i++ {
				uid := restored[i]
				for !user.GetE2E().HasAuthenticatedChannel(uid) {
					time.Sleep(time.Second)
				}
				jww.INFO.Printf("[UD] Authenticated channel established for %s", uid)
			}
		}
		usernameSearchStr := viper.GetString(udSearchUsernameFlag)
		emailSearchStr := viper.GetString(udSearchEmailFlag)
		phoneSearchStr := viper.GetString(udSearchPhoneFlag)

		var facts fact.FactList
		if usernameSearchStr != "" {
			f, err := fact.NewFact(fact.Username, usernameSearchStr)
			if err != nil {
				jww.FATAL.Panicf("Failed to create new fact: %+v", err)
			}
			facts = append(facts, f)
		}
		if emailSearchStr != "" {
			f, err := fact.NewFact(fact.Email, emailSearchStr)
			if err != nil {
				jww.FATAL.Panicf("Failed to create new fact: %+v", err)
			}
			facts = append(facts, f)
		}
		if phoneSearchStr != "" {
			f, err := fact.NewFact(fact.Phone, phoneSearchStr)
			if err != nil {
				jww.FATAL.Panicf("Failed to create new fact: %+v", err)
			}
			facts = append(facts, f)
		}

		userToRemove := viper.GetString(udRemoveFlag)
		if userToRemove != "" {
			f, err := fact.NewFact(fact.Username, userToRemove)
			if err != nil {
				jww.FATAL.Panicf(
					"Failed to create new fact: %+v", err)
			}
			err = userDiscoveryMgr.PermanentDeleteAccount(f)
			if err != nil {
				fmt.Printf("Couldn't remove user %s\n",
					userToRemove)
				jww.FATAL.Panicf(
					"Failed to remove user %s: %+v",
					userToRemove, err)
			}
			fmt.Printf("Removed user from discovery: %s\n",
				userToRemove)
		}

		if len(facts) == 0 {
			err = user.StopNetworkFollower()
			if err != nil {
				jww.WARN.Print(err)
			}
			return
		}

		cb := func(contacts []contact.Contact, err error) {
			if err != nil {
				jww.FATAL.Panicf("%+v", err)
			}
			for _, c := range contacts {
				printContact(c)
			}
		}

		jww.INFO.Printf("[UD] Search: %v", facts)
		_, _, err = ud.Search(user,
			udContact, cb, facts, single.GetDefaultRequestParams())
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		time.Sleep(91 * time.Second)
		err = user.StopNetworkFollower()
		if err != nil {
			jww.WARN.Print(err)
		}
	},
}

// getUdContactInfo is a helper function which retrieves the necessary
// information to contact UD.
func getUdContactInfo(user *xxdk.E2e) (cert, contactFile []byte, address string,
	err error) {
	// Check if we are using an alternate UD with specified flags for contact
	// information
	if viper.IsSet(alternateUdFlag) {
		altUdCertPath := viper.GetString(alternateUdCertFlag)
		address = viper.GetString(altUdAddressFlag)
		altUdContactFilePath := viper.GetString(altUdContactFileFlag)

		cert, err = utils.ReadFile(altUdCertPath)
		if err != nil {
			return nil, nil, "", errors.Errorf(
				"Failed to read alternative UD cert: %+v", err)
		}

		contactFile, err = utils.ReadFile(altUdContactFilePath)
		if err != nil {
			return nil, nil, "", errors.Errorf(
				"Failed to read alternative UD contact file: %+v", err)
		}

		return
	}

	// Retrieve address
	address = user.GetCmix().GetInstance().GetPartialNdf().Get().UDB.Address

	// Retrieve certificate
	cert = []byte(user.GetCmix().GetInstance().GetPartialNdf().Get().UDB.Cert)

	// Retrieve ID
	udIdData := user.GetCmix().GetInstance().GetPartialNdf().Get().UDB.ID
	udId, err := id.Unmarshal(udIdData)
	if err != nil {
		return nil, nil, "", err
	}

	// Retrieve DH Pub Key
	udDhPubKeyData := user.GetCmix().GetInstance().GetPartialNdf().Get().UDB.DhPubKey
	udDhPubKey := user.GetE2E().GetGroup().NewInt(1)
	err = udDhPubKey.UnmarshalJSON(udDhPubKeyData)
	if err != nil {
		return nil, nil, "", err
	}

	// Construct contact
	udContact := contact.Contact{
		ID:       udId,
		DhPubKey: udDhPubKey,
	}

	contactFile = udContact.Marshal()

	return
}

func init() {
	// User Discovery subcommand Options
	udCmd.Flags().StringP(udRegisterFlag, "r", "",
		"Register this user with user discovery.")
	bindFlagHelper(udRegisterFlag, udCmd)

	udCmd.Flags().StringP(udRemoveFlag, "", "",
		"Remove this user with user discovery.")
	bindFlagHelper(udRemoveFlag, udCmd)

	udCmd.Flags().String(udAddPhoneFlag, "",
		"Add phone number to existing user registration.")
	bindFlagHelper(udAddPhoneFlag, udCmd)

	udCmd.Flags().StringP(udAddEmailFlag, "e", "",
		"Add email to existing user registration.")
	bindFlagHelper(udAddEmailFlag, udCmd)

	udCmd.Flags().String(udConfirmFlag, "", "Confirm fact with confirmation ID.")
	bindFlagHelper(udConfirmFlag, udCmd)

	udCmd.Flags().StringP(udLookupFlag, "u", "",
		"Look up user ID. Use '0x' or 'b64:' for hex and base64 representations.")
	bindFlagHelper(udLookupFlag, udCmd)

	udCmd.Flags().String(udSearchUsernameFlag, "",
		"Search for users with this username.")
	bindFlagHelper(udSearchUsernameFlag, udCmd)

	udCmd.Flags().String(udSearchEmailFlag, "",
		"Search for users with this email address.")
	bindFlagHelper(udSearchEmailFlag, udCmd)

	udCmd.Flags().String(udSearchPhoneFlag, "",
		"Search for users with this email address.")
	bindFlagHelper(udSearchPhoneFlag, udCmd)

	udCmd.Flags().String(udBatchAddFlag, "",
		"Path to JSON marshalled slice of partner IDs that will be looked up on UD.")
	bindFlagHelper(udBatchAddFlag, udCmd)

	rootCmd.AddCommand(udCmd)
}
