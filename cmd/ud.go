///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// Package cmd initializes the CLI and config parsers as well as the logger.
package cmd

import (
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/client/interfaces/contact"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/switchboard"
	"gitlab.com/elixxir/client/ud"
	"gitlab.com/elixxir/primitives/fact"
)

// udCmd user discovery subcommand, allowing user lookup and registration for
// allowing others to search.
// This basically runs a client for these functions with the UD module enabled.
// Normally, clients don't need it so it is not loaded for the rest of the
// commands.
var udCmd = &cobra.Command{
	Use: "ud",
	Short: ("Register for & search users using the xxnet user discovery " +
		"service"),
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		client := initClient()
		user := client.GetUser()
		jww.INFO.Printf("User: %s", user.ID)
		writeContact(user.GetContact())

		userDiscoveryMgr, err := ud.NewManager(client)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		// Set up reception handler
		swboard := client.GetSwitchboard()
		recvCh := make(chan message.Receive, 10000)
		listenerID := swboard.RegisterChannel("DefaultCLIReceiver",
			switchboard.AnyUser(), message.Text, recvCh)
		jww.INFO.Printf("Message ListenerID: %v", listenerID)

		// Set up auth request handler, which simply prints the
		// user id of the requestor.
		authMgr := client.GetAuthRegistrar()
		authMgr.AddGeneralRequestCallback(printChanRequest)

		// If unsafe channels, add auto-acceptor
		if viper.GetBool("unsafe-channel-creation") {
			authMgr.AddGeneralRequestCallback(func(
				requestor contact.Contact, message string) {
				jww.INFO.Printf("Got Request: %s", requestor.ID)
				err := client.ConfirmAuthenticatedChannel(
					requestor)
				if err != nil {
					jww.FATAL.Panicf("%+v", err)
				}
			})
		}

		err := client.StartNetworkFollower()
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		// Wait until connected or crash on timeout
		connected := make(chan bool, 10)
		client.GetHealth().AddChannel(connected)
		waitUntilConnected(connected)

		userToRegister := viper.GetString("register")
		if userToRegister != "" {
			err = userDiscoveryMgr.Register(userToRegister)
			if err != nil {
				jww.FATAL.Panicf("%+v", err)
			}
		}

		var facts []fact.Fact
		phone := viper.GetString("addphone")
		if phone != "" {
			f, err := fact.NewFact(fact.Phone, phone)
			if err != nil {
				jww.FATAL.Panicf("%+v", err)
			}
			facts = append(facts, f)
		}
		email := viper.GetString("addemail")
		if email != "" {
			f, err := fact.NewFact(fact.Email, email)
			if err != nil {
				jww.FATAL.Panicf("%+v", err)
			}
			facts = append(facts, f)
		}

		for i := 0; i < len(facts); i++ {
			r, err := userDiscoveryMgr.SendRegisterFact(facts[i])
			if err != nil {
				jww.FATAL.Panicf("%+v", err)
			}
			// TODO Store the code

			fmt.Printf("Fact Add Response: %+v", r)
		}

		confirmID := viper.GetString("confirm")
		if confirmID != "" {
			// TODO Lookup code
			err = userDiscoveryMgr.SendConfirmFact(confirmID,
				confirmID)
			if err != nil {
				jww.FATAL.Panicf("%+v", err)
			}
		}

		lookupIDStr := viper.GetString("lookup")
		if lookupIDStr != "" {
			lookupID, ok := parseRecipient(lookupIDStr)
			if !ok {
				jww.FATAL.Panicf("Could not parse: %s",
					lookupIDStr)
			}
			userDiscoveryMgr.Lookup(lookupID,
				func(newContact contact.Contact, err Error) {
					if err != nil {
						jww.FATAL.Panicf("%+v", err)
					}
					fmt.Printf(newContact.String())
				},
				time.Duration(10*time.Second))
			time.Sleep(11 * time.Second)
		}

		usernameSrchStr := viper.GetString("searchusername")
		emailSrchStr := viper.GetSTring("searchemail")
		phoneSrchStr := viper.GetSTring("searchphone")

		var facts FactList
		if usernameSrchStr != "" {
			f, err := fact.NewFact(fact.Username, usernameSrchStr)
			if err != nil {
				jww.FATAL.Panicf("%+v", err)
			}
			facts = append(facts, f)
		}
		if emailSrchStr != "" {
			f, err := fact.NewFact(fact.Email, emailSrchStr)
			if err != nil {
				jww.FATAL.Panicf("%+v", err)
			}
			facts = append(facts, f)
		}
		if phoneSrchStr != "" {
			f, err := fact.NewFact(fact.Phone, phoneSrchStr)
			if err != nil {
				jww.FATAL.Panicf("%+v", err)
			}
			facts = append(facts, f)
		}

		if len(facts) == 0 {
			client.StopNetworkFollowers(10 * time.Second)
			return
		}

		err := userDiscoveryMgr.Search(facts,
			func(contacts []contact.Contact, err error) {
				if err != nil {
					jww.FATAL.Panicf("%+v", err)
				}
				for i := 0; i < len(contacts); i++ {
					fmt.Printf(contacts[i].String())
				}
			}, 10*time.Second)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}
		time.Sleep(11 * time.Second)
		client.StopNetworkFollowers(10 * time.Second)
	},
}

func init() {
	// User Discovery subcommand Options
	udCmd.Flags().StringP("register", "r", "",
		"Register this user with user discovery")
	viper.BindPFlag("register",
		udCmd.Flags().Lookup("register"))
	udCmd.Flags().StringP("addphone", "", "",
		"Add phone number to existing user registration.")
	viper.BindPFlag("addphone", udCmd.Flags().Lookup("addphone"))
	udCmd.Flags().StringP("addemail", "e", "",
		"Add email to existing user registration.")
	viper.BindPFlag("addemail", udCmd.Flags().Lookup("addemail"))
	udCmd.Flags().StringP("confirm", "", "",
		"Confirm fact with confirmation id")
	viper.BindPFlag("confirm", udCmd.Flags().Lookup("confirm"))

	udCmd.Flags().StringP("lookup", "u", "",
		"Look up user ID. Use '0x' or 'b64:' for hex and base64 "+
			"representations")
	viper.BindPFlag("lookup", udCmd.Flags().Lookup("lookup"))
	udCmd.Flags().StringP("searchusername", "", "",
		"Search for users with this username")
	viper.BindPFlag("searchusername",
		udCmd.Flags().Lookup("searchusername"))
	udCmd.Flags().StringP("searchemail", "", "",
		"Search for users with this email address")
	viper.BindPFlag("searchemail",
		udCmd.Flags().Lookup("searchemail"))
	udCmd.Flags().StringP("searchphone", "", "",
		"Search for users with this email address")
	viper.BindPFlag("searchphone",
		udCmd.Flags().Lookup("searchphone"))

	rootCmd.AddCommand(udCmd)
}
