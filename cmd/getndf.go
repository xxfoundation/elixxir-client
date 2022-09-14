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
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/comms/client"

	// "gitlab.com/elixxir/crypto/contact"
	// "gitlab.com/elixxir/client/interfaces/message"
	// "gitlab.com/elixxir/client/switchboard"
	// "gitlab.com/elixxir/client/ud"
	// "gitlab.com/elixxir/primitives/fact"
	"gitlab.com/elixxir/client/xxdk"
	"gitlab.com/xx_network/comms/connect"
	//"time"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/xx_network/primitives/id"
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
		if viper.IsSet(ndfEnvFlag) {
			var ndfJSON []byte
			var err error
			switch viper.GetString(ndfEnvFlag) {
			case mainnet:
				ndfJSON, err = xxdk.DownloadAndVerifySignedNdfWithUrl(mainNetUrl, mainNetCert)
				if err != nil {
					jww.FATAL.Panicf(err.Error())
				}
			case release:
				ndfJSON, err = xxdk.DownloadAndVerifySignedNdfWithUrl(releaseUrl, releaseCert)
				if err != nil {
					jww.FATAL.Panicf(err.Error())
				}

			case dev:
				ndfJSON, err = xxdk.DownloadAndVerifySignedNdfWithUrl(devUrl, devCert)
				if err != nil {
					jww.FATAL.Panicf(err.Error())
				}
			case testnet:
				ndfJSON, err = xxdk.DownloadAndVerifySignedNdfWithUrl(testNetUrl, testNetCert)
				if err != nil {
					jww.FATAL.Panicf(err.Error())
				}
			default:
				jww.FATAL.Panicf("env flag with unknown flag (%s)",
					viper.GetString(ndfEnvFlag))
			}
			// Print to stdout
			fmt.Printf("%s", ndfJSON)
		} else {

			// Note: getndf prints to stdout, so we default to not do that
			logLevel := viper.GetUint(logLevelFlag)
			logPath := viper.GetString(logFlag)
			if logPath == "-" || logPath == "" {
				logPath = "getndf.log"
			}
			initLog(logLevel, logPath)
			jww.INFO.Printf(Version())
			gwHost := viper.GetString(ndfGwHostFlag)
			permHost := viper.GetString(ndfPermHostFlag)
			certPath := viper.GetString(ndfCertFlag)

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

			// Gateway lookup
			if gwHost != "" {
				ndfJSon, err := xxdk.DownloadNdfFromGateway(gwHost, cert)
				if err != nil {
					jww.FATAL.Panicf("%v", err)
				}
				fmt.Printf("%s", ndfJSon)
				return
			}

			if permHost != "" {
				// Establish parameters for gRPC
				params := connect.GetDefaultHostParams()
				params.AuthEnabled = false

				// Construct client's gRPC comms object
				comms, _ := client.NewClientComms(nil, nil, nil, nil)

				// Establish host for scheduling server
				host, _ := connect.NewHost(&id.Permissioning, permHost,
					cert, params)

				// Construct a dummy message
				pollMsg := &pb.NDFHash{
					Hash: []byte("DummyUserRequest"),
				}

				// Send request to scheduling and get response
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
	getNDFCmd.Flags().StringP(ndfGwHostFlag, "", "",
		"Poll this gateway host:port for the NDF")
	bindFlagHelper(ndfGwHostFlag, getNDFCmd)

	getNDFCmd.Flags().StringP(ndfPermHostFlag, "", "",
		"Poll this registration host:port for the NDF")
	bindFlagHelper(ndfPermHostFlag, getNDFCmd)

	getNDFCmd.Flags().StringP(ndfCertFlag, "", "",
		"Check with the TLS certificate at this path")
	bindFlagHelper(ndfCertFlag, getNDFCmd)

	getNDFCmd.Flags().StringP(ndfEnvFlag, "", "",
		"Downloads and verifies a signed NDF from a specified environment. "+
			"Accepted environment flags include mainnet, release, testnet, and dev")
	bindFlagHelper(ndfEnvFlag, getNDFCmd)

	rootCmd.AddCommand(getNDFCmd)
}
