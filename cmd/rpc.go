////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

// The group subcommand allows creation and sending messages to groups

package cmd

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"

	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/collective/versioned"
	"gitlab.com/elixxir/client/v4/rpc"
	"gitlab.com/elixxir/client/v4/xxdk"
	"gitlab.com/elixxir/crypto/nike"
	"gitlab.com/elixxir/crypto/nike/ecdh"
	"gitlab.com/xx_network/primitives/id"
)

// DM Specific command line options
const (
	rpcServerPubFlag  = "rpcPubkey"
	rpcServerIDFlag   = "rpcID"
	rpcEchoServerFlag = "rpcEchoServer"
	rpcEchoSizeFlag   = "rpcEchoSize"
)

// groupCmd represents the base command when called without any subcommands
var rpcCmd = &cobra.Command{
	Use:   "rpc",
	Short: "RPC commands for cMix client",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		initLog(viper.GetUint(logLevelFlag), viper.GetString(logFlag))
		cmixParams, _ := initParams()
		user := loadOrInitCmix([]byte(viper.GetString(passwordFlag)),
			viper.GetString(sessionFlag), "", cmixParams)

		jww.INFO.Printf("Starting Network followers...")

		err := user.StartNetworkFollower(5 * time.Second)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		jww.INFO.Printf("Network followers started!")

		connected := make(chan bool, 10)
		user.GetCmix().AddHealthCallback(
			func(isConnected bool) {
				connected <- isConnected
			})
		waitUntilConnected(connected)

		// Run an echo Server and exit when it exits
		if viper.GetBool(rpcEchoServerFlag) {
			rpcEchoServer(user)
			return
		}

		startTime := time.Now()
		// Send a message to an xxRPC server
		serverKey, serverID, ok := getServerInfo()

		if !ok {
			fmt.Printf("Couldn't parse server info (%s, %s)...",
				rpcServerPubFlag, rpcServerIDFlag)
			os.Exit(-1)
		}

		request := []byte(viper.GetString(messageFlag))
		res := rpc.Send(user.GetCmix(), serverID,
			serverKey, request,
			cmix.GetDefaultCMIXParams())

		res.Callback(
			func(response []byte) {
				var prettyJSON bytes.Buffer
				err := json.Indent(&prettyJSON, response,
					"", "\t")
				if err != nil {
					jww.FATAL.Panicf("%s: %+v",
						response, err)
				}
				fmt.Printf("Response (%s): %s",
					time.Since(startTime),
					prettyJSON.String())
			},
			func(err error) {
				jww.ERROR.Printf("Error: %+v", err)
			})

		returnMsg := res.Wait()

		fmt.Printf("Response Message: %s\n", returnMsg)
		fmt.Printf("Time to receive: %s\n", time.Since(startTime))
	},
}

func loadRPCServer(kv versioned.KV) (*id.ID, nike.PrivateKey, error) {
	serverIDObj, err := kv.Get("rpcServerID", 0)
	if err != nil {
		return nil, nil, err
	}
	serverID, err := id.Unmarshal(serverIDObj.Data)
	if err != nil {
		return nil, nil, err
	}

	serverKeyObj, err := kv.Get("rpcServerKey", 0)
	if err != nil {
		return nil, nil, err
	}
	serverKey := ecdh.ECDHNIKE.NewEmptyPrivateKey()
	err = serverKey.FromBytes(serverKeyObj.Data)
	if err != nil {
		return nil, nil, err
	}

	return serverID, serverKey, err
}

func saveRPCServer(serverID *id.ID, serverKey nike.PrivateKey,
	kv versioned.KV) error {
	serverIDObj := &versioned.Object{
		Version:   0,
		Timestamp: time.Now(),
		Data:      serverID.Bytes(),
	}
	err := kv.Set("rpcServerID", serverIDObj)
	if err != nil {
		return err
	}
	serverKeyObj := &versioned.Object{
		Version:   0,
		Timestamp: time.Now(),
		Data:      serverKey.Bytes(),
	}
	return kv.Set("rpcServerKey", serverKeyObj)
}

func newRPCServer(net *xxdk.Cmix) (*id.ID, nike.PrivateKey) {
	rng := net.GetCmix().RNGStreamGenerator().GetStream()
	defer rng.Close()
	serverID, err := rpc.GenerateRandomID(net.GetCmix())
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	serverPriv, serverPub := ecdh.ECDHNIKE.NewKeypair(rng)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}

	jww.INFO.Printf("[RPC] GENKEYS: %s, %s",
		base64.RawStdEncoding.EncodeToString(serverPriv.Bytes()),
		base64.RawStdEncoding.EncodeToString(serverPub.Bytes()))

	return serverID, serverPriv
}

func rpcEchoServer(net *xxdk.Cmix) {
	kv := net.GetStorage().GetKV()
	var srvID *id.ID
	var srvPriv nike.PrivateKey
	srvID, srvPriv, err := loadRPCServer(kv)
	if err != nil {
		jww.ERROR.Printf("Error loading RPC Server Info: %+v", err)
		srvID, srvPriv = newRPCServer(net)
		saveRPCServer(srvID, srvPriv, kv)
	}
	srvPub := ecdh.ECDHNIKE.DerivePublicKey(srvPriv)

	respSz := viper.GetUint32(rpcEchoSizeFlag)

	echoFn := func(id *id.ID, request []byte) []byte {
		sz := respSz
		if sz == 0 {
			sz = uint32(len(request))
		}
		reply := make([]byte, sz)
		for i := 0; i < len(reply); i++ {
			reply[i] = request[i%len(request)]
		}
		return reply
	}

	jww.INFO.Printf("RPCSERVERID: %s",
		base64.RawStdEncoding.EncodeToString(srvID.Bytes()))
	jww.INFO.Printf("RPCSERVERKEY: %s, %v",
		base64.RawStdEncoding.EncodeToString(srvPub.Bytes()), srvPub.Bytes())

	// pubKey := ecdh.ECDHNIKE.NewEmptyPublicKey()
	// err = pubKey.FromBytes(srvPub.Bytes())
	// if err != nil {
	// 	jww.WARN.Panicf("unable to decode server reception key: %+v",
	// 		err)
	// }

	server := rpc.NewServer(net.GetCmix(), echoFn, srvID,
		srvPriv)
	server.Start()
	select {}
}

func init() {
	rpcCmd.Flags().BoolP(rpcEchoServerFlag, "d", false,
		"Run an echo server")
	viper.BindPFlag(rpcEchoServerFlag,
		rpcCmd.Flags().Lookup(rpcEchoServerFlag))
	rpcCmd.Flags().Uint32P(rpcEchoSizeFlag, "r", 8192,
		"Size of the repeated message, the echo server will copy "+
			"the request until this sized message is achieved. "+
			"The default is 0 which means to copy the request once")
	viper.BindPFlag(rpcEchoSizeFlag,
		rpcCmd.Flags().Lookup(rpcEchoSizeFlag))

	rpcCmd.Flags().StringP(rpcServerPubFlag, "k", "",
		"The server's public key (base64)")
	viper.BindPFlag(rpcServerPubFlag,
		rpcCmd.Flags().Lookup(rpcServerPubFlag))
	rpcCmd.Flags().StringP(rpcServerIDFlag, "i", "",
		"The server's reception ID (base64)")
	viper.BindPFlag(rpcServerIDFlag,
		rpcCmd.Flags().Lookup(rpcServerIDFlag))

	rootCmd.AddCommand(rpcCmd)
}

func getServerInfo() (nike.PublicKey, *id.ID, bool) {
	pubBytesStr := viper.GetString(rpcServerPubFlag)
	pubBytes, err := base64.RawStdEncoding.DecodeString(pubBytesStr)
	if err != nil {
		jww.WARN.Printf("unable to read server public key: %+v",
			err)
		return nil, nil, false
	}
	jww.INFO.Printf("RPC PUBBYTES: %v", pubBytes)
	pubKey := ecdh.ECDHNIKE.NewEmptyPublicKey()
	err = pubKey.FromBytes(pubBytes)
	if err != nil {
		jww.WARN.Printf("unable to decode server public key: %+v",
			err)
		return nil, nil, false
	}

	serverIDStr := viper.GetString(rpcServerIDFlag)
	serverIDBytes, err := base64.RawStdEncoding.DecodeString(serverIDStr)
	if err != nil {
		jww.WARN.Printf("unable to read server reception id: %+v",
			err)
		return nil, nil, false
	}
	serverID, err := id.Unmarshal(serverIDBytes)
	if err != nil {
		jww.WARN.Printf("unable to decode server reception id: %+v",
			err)
		return nil, nil, false
	}

	return pubKey, serverID, true
}
