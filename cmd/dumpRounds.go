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
	"strconv"
	"time"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/cmix/rounds"
	"gitlab.com/xx_network/comms/signature"
	"gitlab.com/xx_network/crypto/signature/ec"
	"gitlab.com/xx_network/primitives/id"
)

// singleCmd is the single-use subcommand that allows for sending and responding
// to single-use messages.
var dumpRoundsCmd = &cobra.Command{
	Use:   "dumprounds",
	Short: "Dump round information for specified rounds",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		roundIDs := parseRoundIDs(args)

		cmixParams, e2eParams := initParams()
		client := initE2e(cmixParams, e2eParams)

		err := client.StartNetworkFollower(5 * time.Second)
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

		numRequests := len(roundIDs)
		requestCh := make(chan bool, numRequests)

		ecp := client.GetStorage().GetNDF().Registration.EllipticPubKey
		fmt.Printf("pubkey: %s\n\n", ecp)
		pubkey, err := ec.LoadPublicKey(ecp)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		fmt.Printf("pubkey unserialized: %s\n\n", pubkey.MarshalText())

		rcb := func(round rounds.Round, success bool) {
			fmt.Printf("Lookup for %v: %v\n\n", round.ID, success)
			fmt.Printf("Info: %v\n\n", round)

			ri := round.Raw
			err = signature.VerifyEddsa(ri, pubkey)
			if err != nil {
				jww.FATAL.Panicf("%+v", err)
			}
			requestCh <- success
		}

		for i := range roundIDs {
			rid := roundIDs[i]
			err := client.GetCmix().LookupHistoricalRound(rid, rcb)
			if err != nil {
				fmt.Printf("error on %v: %v", rid, err)
			}
		}

		for done := 0; done < numRequests; done++ {
			res := <-requestCh
			fmt.Printf("request complete: %v", res)
		}
	},
}

func init() {
	rootCmd.AddCommand(dumpRoundsCmd)
}

func parseRoundIDs(roundStrs []string) []id.Round {
	var roundIDs []id.Round
	for _, r := range roundStrs {
		i, err := parseRoundID(r)
		if err != nil {
			fmt.Printf("Could not parse into round ID: %s, %v",
				r, err)
		} else {
			roundIDs = append(roundIDs, id.Round(i))
		}
	}
	return roundIDs
}

func parseRoundID(roundStr string) (uint64, error) {
	return strconv.ParseUint(roundStr, 10, 64)
}
