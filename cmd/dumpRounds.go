////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

// Package cmd initializes the CLI and config parsers as well as the logger.
package cmd

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/viper"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/client/v5/cmix/rounds"
	"gitlab.com/elixxir/client/v5/xxdk"
	"gitlab.com/xx_network/comms/signature"
	"gitlab.com/xx_network/crypto/signature/ec"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
)

// dumpRoundsCmd allows the user to view network information about a specific
// round on the network.
var dumpRoundsCmd = &cobra.Command{
	Use:   "dumprounds",
	Short: "Dump round information for specified rounds",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		roundIDs := parseRoundIDs(args)

		cmixParams, e2eParams := initParams()
		authCbs := makeAuthCallbacks(
			viper.GetBool(unsafeChannelCreationFlag), e2eParams)
		user := initE2e(cmixParams, e2eParams, authCbs)
		err := user.StartNetworkFollower(5 * time.Second)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		connected := make(chan bool, 10)
		user.GetCmix().AddHealthCallback(
			func(isconnected bool) {
				connected <- isconnected
			})
		waitUntilConnected(connected)

		roundInfos := dumpRounds(roundIDs, user)

		ndf := user.GetStorage().GetNDF()

		for i := range roundInfos {
			printRoundInfo(roundInfos[i])
			printAndVerifyRoundSig(roundInfos[i], ndf)
		}
	},
}

func init() {
	rootCmd.AddCommand(dumpRoundsCmd)
}

func printRoundInfo(round rounds.Round) {
	fmt.Printf("Round %v:", round.ID)
	fmt.Printf("\n\tBatch size: %v, State: %v",
		round.BatchSize, round.State)
	fmt.Printf("\n\tUpdateID: %v, AddrSpaceSize: %v",
		round.UpdateID, round.AddressSpaceSize)

	fmt.Printf("\n\tTopology: ")
	for i, nodeId := range round.Raw.Topology {
		nidStr := base64.StdEncoding.EncodeToString(
			nodeId)
		fmt.Printf("\n\t\t%d\t-\t%s", i, nidStr)
	}

	fmt.Printf("\n\tTimestamps: ")
	for state, ts := range round.Timestamps {
		fmt.Printf("\n\t\t%v  \t-\t%v", state, ts)
	}

	fmt.Printf("\n\tErrors (%d): ", len(round.Raw.Errors))
	for i, err := range round.Raw.Errors {
		fmt.Printf("\n\t\t%d - %v", i, err)
	}

	fmt.Printf("\n\tClientErrors (%d): ",
		len(round.Raw.ClientErrors))
	for _, ce := range round.Raw.ClientErrors {
		fmt.Printf("\n\t\t%s - %v, Src: %v",
			base64.StdEncoding.EncodeToString(
				ce.ClientId),
			ce.Error,
			base64.StdEncoding.EncodeToString(
				ce.Source))
	}
}

func printAndVerifyRoundSig(round rounds.Round, ndf *ndf.NetworkDefinition) {
	registration := ndf.Registration
	ecp := registration.EllipticPubKey
	pubkey, err := ec.LoadPublicKey(ecp)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	fmt.Printf("registration pubkey: %s\n\n", pubkey.MarshalText())

	ri := round.Raw
	err = signature.VerifyEddsa(ri, pubkey)
	if err != nil {
		fmt.Printf("\n\tECC signature failed: %v", err)
		fmt.Printf("\n\tuse trace logging for sig details")
	} else {
		fmt.Printf("\n\tECC signature succeeded!\n\n")
	}

	// fmt.Printf("Round Info RAW: %v\n\n", round)

	// rsapubkey, _ := rsa.LoadPublicKeyFromPem([]byte(
	// 	registration.TlsCertificate))
	// signature.VerifyRsa(ri, rsapubkey)
	// if err != nil {
	// 	fmt.Printf("RSA signature failed: %v", err)
	// 	fmt.Printf("use trace logging for sig details")
	// } else {
	// 	fmt.Printf("RSA signature succeeded!")
	// }
}

func dumpRounds(roundIDs []id.Round, user *xxdk.E2e) []rounds.Round {
	numRequests := len(roundIDs)
	requestCh := make(chan rounds.Round, numRequests)

	rcb := func(round rounds.Round, success bool) {
		if !success {
			fmt.Printf("round %v lookup failed", round.ID)
		}
		requestCh <- round
	}

	for i := range roundIDs {
		rid := roundIDs[i]
		err := user.GetCmix().LookupHistoricalRound(rid, rcb)
		if err != nil {
			fmt.Printf("error on %v: %v", rid, err)
		}
	}

	roundInfos := make([]rounds.Round, 0)
	for done := 0; done < numRequests; done++ {
		res := <-requestCh
		roundInfos = append(roundInfos, res)
		jww.DEBUG.Printf("request complete: %v", res)
	}
	return roundInfos
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
