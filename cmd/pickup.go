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
	"time"

	"github.com/pkg/errors"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/client/cmix/pickup"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/crypto/randomness"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/ndf"
)

// pickupCmd allows the user to view network information about a specific
// round on the network.
var pickupCmd = &cobra.Command{
	Use:   "pickup",
	Short: "Download the bloomfilter and messages for a round",
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

		ndf := user.GetStorage().GetNDF()

		gwID := getGatewayID(ndf)
		clientID := parseRecipient(viper.GetString(pickupID))

		// First we get round info, then we use the timestamps to
		// calculate the right ephID and retrieve the right bloom filter
		roundInfos := dumpRounds(roundIDs, user)
		for i := range roundInfos {
			ri := roundInfos[i]
			ephIDs := getEphID(clientID, uint(ri.AddressSpaceSize),
				ri.Timestamps[states.QUEUED])

			for j := range ephIDs {
				ephID := ephIDs[j]
				fmt.Printf("Getting messages for %s, %d\n",
					ri.ID, ephID.Id.Int64())
				msgRsp, err := getMessagesFromRound(gwID, ri.ID,
					ephID.Id,
					user.GetComms())
				if err != nil {
					fmt.Printf("\n\nround pickup: %+v\n\n",
						err)
				}
				fmt.Printf("=====ROUNDPICKUP=====\n\n%+v\n\n\n", msgRsp)
				fmt.Printf("%d messages for user %d", len(msgRsp.Messages), ephIDs)
				for k := range msgRsp.Messages {
					fmt.Printf("%v\n", msgRsp.Messages[k].PayloadA)
				}
			}
		}
	},
}

func init() {
	pickupCmd.Flags().StringP(pickupGW, "g", "",
		"gateway (base64 address string) to download from")
	bindFlagHelper(pickupGW, pickupCmd)

	pickupCmd.Flags().StringP(pickupID, "i", "",
		"id to check")
	bindFlagHelper(pickupID, pickupCmd)

	rootCmd.AddCommand(pickupCmd)
}

func getEphID(id *id.ID, addrSize uint,
	roundStart time.Time) []ephemeral.ProtoIdentity {

	fmt.Printf("Getting EphIDs for %s", roundStart)

	ephIDs, err := ephemeral.GetIdsByRange(id,
		addrSize,
		roundStart,
		time.Duration(12*time.Hour))
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}

	if len(ephIDs) == 0 {
		jww.FATAL.Panicf("No ephemeral ids found!")
	}

	return ephIDs
}
func getGatewayID(ndf *ndf.NetworkDefinition) *id.ID {
	gateways := ndf.Gateways
	gwID := viper.GetString(pickupGW)

	if gwID == "" {
		rng := csprng.NewSystemRNG()
		i := randomness.ReadRangeUint32(0, uint32(len(gateways)), rng)
		id, err := id.Unmarshal([]byte(gateways[i].ID))
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}
		fmt.Printf("selected random gw: %s\n", id)
		return id
	}

	for i := range gateways {
		curID, _ := id.Unmarshal(gateways[i].ID)
		jww.DEBUG.Printf("%s ==? %s", gwID, curID)
		if curID.String() == gwID {
			return curID
		}
	}

	jww.FATAL.Panicf("%s is not a gateway in the NDF", gwID)
	return nil
}

func getBloomFilter(targetGW string, ephID int64) *pb.ClientBlooms {
	return nil
}

func getMessagesFromRound(targetGW *id.ID, roundID id.Round,
	ephID ephemeral.Id, comms pickup.MessageRetrievalComms) (
	*pb.GetMessagesResponse, error) {

	host, ok := comms.GetHost(targetGW)
	if !ok {
		return nil, errors.Errorf("can't find host %s", targetGW)
	}
	msgReq := &pb.GetMessages{
		ClientID: ephID[:],
		RoundID:  uint64(roundID),
		Target:   targetGW.Marshal(),
	}

	jww.DEBUG.Printf("Sending request: %+v", msgReq)

	return comms.RequestMessages(host, msgReq)
}
