///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// Package cmd initializes the CLI and config parsers as well as the logger.
package cmd

import (
	"fmt"
	"strings"
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
// with the UD module enabled. Normally, clients do not need it so it is not
// loaded for the rest of the commands.
var udCmd = &cobra.Command{
	Use:   "ud",
	Short: "Register for and search users using the xx network user discovery service.",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmixParams, e2eParams := initParams()
		client := initE2e(cmixParams, e2eParams)

		// get user and save contact to file
		user := client.GetReceptionIdentity()
		jww.INFO.Printf("[UD]User: %s", user.ID)
		writeContact(user.GetContact())

		err := client.StartNetworkFollower(50 * time.Millisecond)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		jww.TRACE.Printf("[UD] Waiting for connection...")

		// Wait until connected or crash on timeout
		connected := make(chan bool, 10)
		client.GetCmix().AddHealthCallback(
			func(isconnected bool) {
				connected <- isconnected
			})
		waitUntilConnected(connected)

		jww.TRACE.Printf("[UD] Connected!")

		// Make user discovery manager
		rng := client.GetRng()
		userToRegister := viper.GetString("register")
		jww.TRACE.Printf("[UD] Registering user %v...", userToRegister)
		userDiscoveryMgr, err := ud.NewManager(client, client.GetComms(),
			client.NetworkFollowerStatus, userToRegister, nil)
		if err != nil {
			if strings.Contains(err.Error(), ud.IsRegisteredErr) {
				userDiscoveryMgr, err = ud.LoadManager(client, client.GetComms())
				if err != nil {
					jww.FATAL.Panicf("Failed to load UD manager: %+v", err)
				}
			} else {
				jww.FATAL.Panicf("Failed to create new UD manager: %+v", err)

			}
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

		udContact, err := userDiscoveryMgr.GetContact()
		if err != nil {
			fmt.Printf("Failed to get user discovery contact object: %+v", err)
			jww.FATAL.Printf("Failed to get user discovery contact object: %+v", err)
		}

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

			stream := rng.GetStream()
			_, _, err = ud.Lookup(client.GetCmix(),
				stream, client.GetE2E().GetGroup(),
				udContact, cb, lookupID, single.GetDefaultRequestParams())
			if err != nil {
				jww.WARN.Printf("Failed UD lookup: %+v", err)
			}
			stream.Close()

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
				idListFile, client, userDiscoveryMgr, nil)
			if err != nil {
				jww.FATAL.Panicf("%+v", err)
			}
			for i := 0; i < len(restored); i++ {
				uid := restored[i]
				for !client.GetE2E().HasAuthenticatedChannel(uid) {
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
			err = client.StopNetworkFollower()
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

		stream := rng.GetStream()
		defer stream.Close()
		jww.INFO.Printf("[UD] Search: %v", facts)
		_, _, err = ud.Search(client.GetCmix(),
			client.GetEventReporter(),
			stream, client.GetE2E().GetGroup(),
			udContact, cb, facts, single.GetDefaultRequestParams())
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		time.Sleep(91 * time.Second)
		err = client.StopNetworkFollower()
		if err != nil {
			jww.WARN.Print(err)
		}
	},
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
