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
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	cmdUtils "gitlab.com/elixxir/client/cmdUtils"

	// "gitlab.com/elixxir/crypto/contact"
	// "gitlab.com/elixxir/client/interfaces/message"
	// "gitlab.com/elixxir/client/switchboard"
	// "gitlab.com/elixxir/client/ud"
	// "gitlab.com/elixxir/primitives/fact"
	"gitlab.com/elixxir/client/xxdk"
	"gitlab.com/elixxir/comms/client"
	"gitlab.com/xx_network/comms/connect"
	//"time"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
	"gitlab.com/xx_network/primitives/utils"
)

const opensslCertDL = "openssl s_client -showcerts -connect ip:port < " +
	"/dev/null 2>&1 | openssl x509 -outform PEM > certfile.pem"

// getNDFCmd user discovery subcommand, allowing user lookup and registration for
// allowing others to search.
// This basically runs a client for these functions with the UD module enabled.
// Normally, clients don't need it so it is not loaded for the rest of the
// commands.
var getNDFCmd = &cobra.Command{
	Use: "getndf",
	Short: "Download the network definition file from the network " +
		"and print it.",
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if viper.IsSet(cmdUtils.NdfEnvFlag) {
			var ndfJSON []byte
			var err error
			switch viper.GetString(cmdUtils.NdfEnvFlag) {
			case cmdUtils.Mainnet:
				ndfJSON, err = xxdk.DownloadAndVerifySignedNdfWithUrl(cmdUtils.MainNetUrl, cmdUtils.MainNetCert)
				if err != nil {
					jww.FATAL.Panicf(err.Error())
				}
			case cmdUtils.Release:
				ndfJSON, err = xxdk.DownloadAndVerifySignedNdfWithUrl(cmdUtils.ReleaseUrl, cmdUtils.ReleaseCert)
				if err != nil {
					jww.FATAL.Panicf(err.Error())
				}

			case cmdUtils.Dev:
				ndfJSON, err = xxdk.DownloadAndVerifySignedNdfWithUrl(cmdUtils.DevUrl, cmdUtils.DevCert)
				if err != nil {
					jww.FATAL.Panicf(err.Error())
				}
			case cmdUtils.Testnet:
				ndfJSON, err = xxdk.DownloadAndVerifySignedNdfWithUrl(cmdUtils.TestNetUrl, cmdUtils.TestNetCert)
				if err != nil {
					jww.FATAL.Panicf(err.Error())
				}
			default:
				jww.FATAL.Panicf("env flag with unknown flag (%s)",
					viper.GetString(cmdUtils.NdfEnvFlag))
			}
			// Print to stdout
			fmt.Printf("%s", ndfJSON)
		} else {

			// Note: getndf prints to stdout, so we default to not do that
			logLevel := viper.GetUint(cmdUtils.LogLevelFlag)
			logPath := viper.GetString(cmdUtils.LogFlag)
			if logPath == "-" || logPath == "" {
				logPath = "getndf.log"
			}
			initLog(logLevel, logPath)
			jww.INFO.Printf(Version())
			gwHost := viper.GetString(cmdUtils.NdfGwHostFlag)
			permHost := viper.GetString(cmdUtils.NdfPermHostFlag)
			certPath := viper.GetString(cmdUtils.NdfCertFlag)

			// Load the certificate
			var cert []byte
			if certPath != "" {
				cert, _ = utils.ReadFile(certPath)
			}
			if len(cert) == 0 {
				jww.FATAL.Panicf("Could not load a certificate, "+
					"provide a certificate file with --cert.\n\n"+
					"You can download a cert using openssl:\n\n%s",
					opensslCertDL)
			}

			params := connect.GetDefaultHostParams()
			params.AuthEnabled = false
			comms, _ := client.NewClientComms(nil, nil, nil, nil)
			// Gateway lookup
			if gwHost != "" {
				host, _ := connect.NewHost(&id.TempGateway, gwHost,
					cert, params)
				dummyID := ephemeral.ReservedIDs[0]
				pollMsg := &pb.GatewayPoll{
					Partial: &pb.NDFHash{
						Hash: nil,
					},
					LastUpdate:    uint64(0),
					ReceptionID:   dummyID[:],
					ClientVersion: []byte(xxdk.SEMVER),
				}
				resp, err := comms.SendPoll(host, pollMsg)
				if err != nil {
					jww.FATAL.Panicf("Unable to poll %s for NDF:"+
						" %+v",
						gwHost, err)
				}
				fmt.Printf("%s", resp.PartialNDF.Ndf)
				return
			}

			if permHost != "" {
				host, _ := connect.NewHost(&id.Permissioning, permHost,
					cert, params)
				pollMsg := &pb.NDFHash{
					Hash: []byte("DummyUserRequest"),
				}
				resp, err := comms.RequestNdf(host, pollMsg)
				if err != nil {
					jww.FATAL.Panicf("Unable to ask %s for NDF:"+
						" %+v",
						permHost, err)
				}
				fmt.Printf("%s", resp.Ndf)
				return
			}

			fmt.Println("Enter --gwhost or --permhost and --cert please")
		}
	},
}

func init() {
	getNDFCmd.Flags().StringP(cmdUtils.NdfGwHostFlag, "", "",
		"Poll this gateway host:port for the NDF")
	cmdUtils.BindFlagHelper(cmdUtils.NdfGwHostFlag, getNDFCmd)

	getNDFCmd.Flags().StringP(cmdUtils.NdfPermHostFlag, "", "",
		"Poll this registration host:port for the NDF")
	cmdUtils.BindFlagHelper(cmdUtils.NdfPermHostFlag, getNDFCmd)

	getNDFCmd.Flags().StringP(cmdUtils.NdfCertFlag, "", "",
		"Check with the TLS certificate at this path")
	cmdUtils.BindFlagHelper(cmdUtils.NdfCertFlag, getNDFCmd)

	getNDFCmd.Flags().StringP(cmdUtils.NdfEnvFlag, "", "",
		"Downloads and verifies a signed NDF from a specified environment. "+
			"Accepted environment flags include mainnet, release, testnet, and dev")
	cmdUtils.BindFlagHelper(cmdUtils.NdfEnvFlag, getNDFCmd)

	rootCmd.AddCommand(getNDFCmd)
}
