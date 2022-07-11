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
		jww.INFO.Printf("User: %s", user.ID)
		writeContact(user.GetContact())

		// // Set up reception handler
		// swBoard := client.GetSwitchboard()
		// recvCh := make(chan message.Receive, 10000)
		// listenerID := swBoard.RegisterChannel("DefaultCLIReceiver",
		// 	switchboard.AnyUser(), message.XxMessage, recvCh)
		// jww.INFO.Printf("Message ListenerID: %v", listenerID)

		// // Set up auth request handler, which simply prints the user ID of the
		// // requester
		// authMgr := client.GetAuthRegistrar()
		// authMgr.AddGeneralRequestCallback(printChanRequest)

		// // If unsafe channels, add auto-acceptor
		// if viper.GetBool(unsafeChannelCreationFlag) {
		// 	authMgr.AddGeneralRequestCallback(func(
		// 		requester contact.Contact) {
		// 		jww.INFO.Printf("Got Request: %s", requester.ID)
		// 		_, err := client.ConfirmAuthenticatedChannel(requester)
		// 		if err != nil {
		// 			jww.FATAL.Panicf("%+v", err)
		// 		}
		// 	})
		// }

		err := client.StartNetworkFollower(50 * time.Millisecond)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		// Wait until connected or crash on timeout
		connected := make(chan bool, 10)
		client.GetCmix().AddHealthCallback(
			func(isconnected bool) {
				connected <- isconnected
			})
		waitUntilConnected(connected)

		// Make user discovery manager
		rng := client.GetRng()
		userToRegister := viper.GetString("register")
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
			r, err := userDiscoveryMgr.SendRegisterFact(newFacts[i])
			if err != nil {
				fmt.Printf("Failed to register fact: %s\n",
					newFacts[i])
				jww.FATAL.Panicf("Failed to send register fact: %+v", err)
			}
			// TODO Store the code?
			jww.INFO.Printf("Fact Add Response: %+v", r)
		}

		confirmID := viper.GetString(udConfirmFlag)
		if confirmID != "" {
			err = userDiscoveryMgr.ConfirmFact(confirmID, confirmID)
			if err != nil {
				fmt.Printf("Couldn't confirm fact: %s\n",
					err.Error())
				jww.FATAL.Panicf("%+v", err)
			}
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
			//if !ok {
			//	jww.FATAL.Panicf("Could not parse recipient: %s", lookupIDStr)
			//}

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
				jww.INFO.Printf("Authenticated channel established for %s", uid)
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
	bindPFlagCheckErr(udRegisterFlag, udCmd)

	udCmd.Flags().StringP(udRemoveFlag, "", "",
		"Remove this user with user discovery.")
	bindPFlagCheckErr(udRemoveFlag, udCmd)

	udCmd.Flags().String(udAddPhoneFlag, "",
		"Add phone number to existing user registration.")
	bindPFlagCheckErr(udAddPhoneFlag, udCmd)

	udCmd.Flags().StringP(udAddEmailFlag, "e", "",
		"Add email to existing user registration.")
	bindPFlagCheckErr(udAddEmailFlag, udCmd)

	udCmd.Flags().String(udConfirmFlag, "", "Confirm fact with confirmation ID.")
	bindPFlagCheckErr(udConfirmFlag, udCmd)

	udCmd.Flags().StringP(udLookupFlag, "u", "",
		"Look up user ID. Use '0x' or 'b64:' for hex and base64 representations.")
	bindPFlagCheckErr(udLookupFlag, udCmd)

	udCmd.Flags().String(udSearchUsernameFlag, "",
		"Search for users with this username.")
	bindPFlagCheckErr(udSearchUsernameFlag, udCmd)

	udCmd.Flags().String(udSearchEmailFlag, "",
		"Search for users with this email address.")
	bindPFlagCheckErr(udSearchEmailFlag, udCmd)

	udCmd.Flags().String(udSearchPhoneFlag, "",
		"Search for users with this email address.")
	bindPFlagCheckErr(udSearchPhoneFlag, udCmd)

	udCmd.Flags().String(udBatchAddFlag, "",
		"Path to JSON marshalled slice of partner IDs that will be looked up on UD.")
	bindPFlagCheckErr(udBatchAddFlag, udCmd)

	rootCmd.AddCommand(udCmd)
}

func printContact(c contact.Contact) {
	jww.DEBUG.Printf("Printing contact: %+v", c)
	cBytes := c.Marshal()
	if len(cBytes) == 0 {
		jww.ERROR.Print("Marshaled contact has a size of 0.")
	} else {
		jww.DEBUG.Printf("Printing marshaled contact of size %d.", len(cBytes))
	}

	fmt.Print(string(cBytes))
}
