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
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"sync"
	"time"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"

	"gitlab.com/elixxir/client/v4/cmix"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/client/v4/collective/versioned"
	"gitlab.com/elixxir/client/v4/dm"
	clientNotif "gitlab.com/elixxir/client/v4/notifications"
	"gitlab.com/elixxir/crypto/codename"
	"gitlab.com/elixxir/crypto/message"
	"gitlab.com/elixxir/crypto/nike/ecdh"
	"gitlab.com/xx_network/primitives/id"
)

// DM Specific command line options
const (
	dmPartnerPubKeyFlag = "dmPubkey"
	dmPartnerTokenFlag  = "dmToken"
)

// groupCmd represents the base command when called without any subcommands
var dmCmd = &cobra.Command{
	Use:   "dm",
	Short: "Group commands for cMix client",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		initLog(viper.GetUint(logLevelFlag), viper.GetString(logFlag))
		cmixParams, _ := initParams()
		user := loadOrInitCmix([]byte(viper.GetString(passwordFlag)),
			viper.GetString(sessionFlag), "", cmixParams)

		// Print user's reception ID
		identity := user.GetStorage().GetReceptionID()
		jww.INFO.Printf("User: %s", identity)

		// NOTE: DM ID's are not storage backed, so we do the
		// storage here.
		ekv := user.GetStorage().GetKV()
		rng := user.GetRng().GetStream()
		defer rng.Close()
		dmIDObj, err := ekv.Get("dmID", 0)
		if err != nil && ekv.Exists(err) {
			jww.FATAL.Panicf("%+v", err)
		}
		var dmID codename.PrivateIdentity
		if ekv.Exists(err) {
			dmID, err = codename.UnmarshalPrivateIdentity(
				dmIDObj.Data)
		} else {
			dmID, err = codename.GenerateIdentity(rng)
		}
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}
		dmToken := dmID.GetDMToken()
		pubKeyBytes := dmID.PubKey[:]

		ekv.Set("dmID", &versioned.Object{
			Version:   0,
			Timestamp: time.Now(),
			Data:      dmID.Marshal(),
		})

		jww.INFO.Printf("DMPUBKEY: %s",
			base64.RawStdEncoding.EncodeToString(pubKeyBytes))
		jww.INFO.Printf("DMTOKEN: %d", dmToken)

		partnerPubKey, partnerDMToken, ok := getDMPartner()
		if !ok {
			jww.WARN.Printf("Setting dm destination to self")
			partnerPubKey = dmID.PubKey
			partnerDMToken = dmToken
		}

		jww.INFO.Printf("DMRECVPUBKEY: %s",
			base64.RawStdEncoding.EncodeToString(partnerPubKey))
		jww.INFO.Printf("DMRECVTOKEN: %d", partnerDMToken)

		recvCh := make(chan message.ID, 10)
		myReceiver := &receiver{
			recv:    recvCh,
			msgData: make(map[message.ID]*msgInfo),
			uuid:    0,
		}
		myNickMgr := dm.NewNicknameManager(identity, ekv)

		sendTracker := dm.NewSendTracker(ekv)

		// Construct notifications manager
		sig := user.GetStorage().GetTransmissionRegistrationValidationSignature()
		nm := clientNotif.NewOrLoadManager(user.GetTransmissionIdentity(), sig,
			user.GetStorage().GetKV(), &clientNotif.MockComms{}, user.GetRng())

		dmClient, err := dm.NewDMClient(&dmID, myReceiver, sendTracker,
			myNickMgr, nm, user.GetCmix(), ekv, user.GetRng(), nil)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}

		err = user.StartNetworkFollower(5 * time.Second)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}
		// Wait until connected or crash on timeout
		connected := make(chan bool, 10)
		user.GetCmix().AddHealthCallback(
			func(isConnected bool) {
				connected <- isConnected
			})
		waitUntilConnected(connected)
		waitForRegistration(user, 0.85)

		msgID, rnd, ephID, err := dmClient.SendText(partnerPubKey,
			partnerDMToken,
			viper.GetString(messageFlag),
			cmix.GetDefaultCMIXParams())
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}
		jww.INFO.Printf("DM Send: %v, %v, %v", msgID, rnd, ephID)

		// Message Reception Loop
		waitTime := viper.GetDuration(waitTimeoutFlag) * time.Second
		maxReceiveCnt := viper.GetInt(receiveCountFlag)
		receiveCnt := 0
		for done := false; !done; {
			if maxReceiveCnt != 0 && receiveCnt >= maxReceiveCnt {
				done = true
				continue
			}
			select {
			case <-time.After(waitTime):
				done = true
			case m := <-recvCh:
				msg := myReceiver.msgData[m]
				selfStr := "Partner"
				if dmID.GetDMToken() == msg.dmToken {
					selfStr = "Self"
					if !bytes.Equal(dmID.PubKey[:],
						msg.partnerKey[:]) {
						jww.FATAL.Panicf(
							"pubkey mismatch!\n")
					}
				}
				fmt.Printf("Message received (%s, %s): %s\n",
					selfStr, msg.mType, msg.content)
				jww.INFO.Printf("Message received: %s\n", msg)
				jww.INFO.Printf("RECVDMPUBKEY: %s",
					base64.RawStdEncoding.EncodeToString(
						msg.partnerKey[:]))
				jww.INFO.Printf("RECVDMTOKEN: %d", msg.dmToken)
				receiveCnt++
			}
		}
		if maxReceiveCnt == 0 {
			maxReceiveCnt = receiveCnt
		}
		fmt.Printf("Received %d/%d messages\n", receiveCnt,
			maxReceiveCnt)

		err = user.StopNetworkFollower()
		if err != nil {
			jww.WARN.Printf(
				"Failed to cleanly close threads: %+v\n",
				err)
		}
		jww.INFO.Printf("Client exiting!")
	},
}

func init() {
	dmCmd.Flags().StringP(dmPartnerPubKeyFlag, "d", "",
		"The public key of the dm partner (base64)")
	viper.BindPFlag(dmPartnerPubKeyFlag, dmCmd.Flags().Lookup(
		dmPartnerPubKeyFlag))

	dmCmd.Flags().StringP(dmPartnerTokenFlag, "t", "",
		"The token of the dm partner (base64)")
	viper.BindPFlag(dmPartnerTokenFlag, dmCmd.Flags().Lookup(
		dmPartnerTokenFlag))

	rootCmd.AddCommand(dmCmd)
}

func getDMPartner() (ed25519.PublicKey, uint32, bool) {
	pubBytesStr := viper.GetString(dmPartnerPubKeyFlag)
	pubBytes, err := base64.RawStdEncoding.DecodeString(pubBytesStr)
	if err != nil {
		jww.WARN.Printf("unable to read partner public key: %+v",
			err)
		return nil, 0, false
	}
	pubKey, err := ecdh.ECDHNIKE.UnmarshalBinaryPublicKey(pubBytes)
	if err != nil {
		jww.WARN.Printf("unable to decode partner public key: %+v",
			err)
		return nil, 0, false
	}
	token := viper.GetUint32(dmPartnerTokenFlag)
	return ecdh.EcdhNike2EdwardsPublicKey(pubKey), token, true
}

type nickMgr struct{}

func (nm *nickMgr) GetNickname(id *id.ID) (string, bool) {
	return base64.RawStdEncoding.EncodeToString(id[:]), true
}

type msgInfo struct {
	messageID  message.ID
	replyID    message.ID
	nickname   string
	content    string
	partnerKey ed25519.PublicKey
	senderKey  ed25519.PublicKey
	dmToken    uint32
	codeset    uint8
	timestamp  time.Time
	round      rounds.Round
	mType      dm.MessageType
	status     dm.Status
	uuid       uint64
}

func (mi *msgInfo) String() string {
	return fmt.Sprintf("[%v-%v] %s: %s", mi.messageID, mi.replyID,
		mi.nickname, mi.content)
}

type receiver struct {
	recv    chan message.ID
	msgData map[message.ID]*msgInfo
	uuid    uint64
	sync.Mutex
}

func (r *receiver) receive(messageID message.ID, replyID message.ID,
	nickname, text string, partnerKey, senderKey ed25519.PublicKey,
	dmToken uint32,
	codeset uint8, timestamp time.Time,
	round rounds.Round, mType dm.MessageType, status dm.Status) uint64 {
	r.Lock()
	defer r.Unlock()
	msg, ok := r.msgData[messageID]
	if !ok {
		r.uuid += 1
		msg = &msgInfo{
			messageID:  messageID,
			replyID:    replyID,
			nickname:   nickname,
			content:    text,
			partnerKey: partnerKey,
			senderKey:  senderKey,
			dmToken:    dmToken,
			codeset:    codeset,
			timestamp:  timestamp,
			round:      round,
			mType:      mType,
			status:     status,
			uuid:       r.uuid,
		}
		r.msgData[messageID] = msg
	} else {
		msg.status = status
	}
	go func() { r.recv <- messageID }()
	return msg.uuid
}

func (r *receiver) Receive(messageID message.ID,
	nickname string, text []byte, partnerKey, senderKey ed25519.PublicKey,
	dmToken uint32,
	codeset uint8, timestamp time.Time,
	round rounds.Round, mType dm.MessageType, status dm.Status) uint64 {
	jww.INFO.Printf("Receive: %v", messageID)
	return r.receive(messageID, message.ID{}, nickname, string(text),
		partnerKey, senderKey, dmToken, codeset, timestamp, round, mType, status)
}

func (r *receiver) ReceiveText(messageID message.ID,
	nickname, text string, partnerKey, senderKey ed25519.PublicKey,
	dmToken uint32,
	codeset uint8, timestamp time.Time,
	round rounds.Round, status dm.Status) uint64 {
	jww.INFO.Printf("ReceiveText: %v", messageID)
	return r.receive(messageID, message.ID{}, nickname, text,
		partnerKey, senderKey, dmToken, codeset, timestamp, round,
		dm.TextType, status)
}
func (r *receiver) ReceiveReply(messageID message.ID,
	reactionTo message.ID, nickname, text string,
	partnerKey, senderKey ed25519.PublicKey, dmToken uint32, codeset uint8,
	timestamp time.Time, round rounds.Round,
	status dm.Status) uint64 {
	jww.INFO.Printf("ReceiveReply: %v", messageID)
	return r.receive(messageID, reactionTo, nickname, text,
		partnerKey, senderKey, dmToken, codeset, timestamp, round,
		dm.TextType, status)
}
func (r *receiver) ReceiveReaction(messageID message.ID,
	reactionTo message.ID, nickname, reaction string,
	partnerKey, senderKey ed25519.PublicKey, dmToken uint32, codeset uint8,
	timestamp time.Time, round rounds.Round,
	status dm.Status) uint64 {
	jww.INFO.Printf("ReceiveReaction: %v", messageID)
	return r.receive(messageID, reactionTo, nickname, reaction,
		partnerKey, senderKey, dmToken, codeset, timestamp, round,
		dm.ReactionType,
		status)
}
func (r *receiver) UpdateSentStatus(uuid uint64, messageID message.ID,
	timestamp time.Time, round rounds.Round, status dm.Status) {
	r.Lock()
	defer r.Unlock()
	jww.INFO.Printf("UpdateSentStatus: %v", messageID)
	msg, ok := r.msgData[messageID]
	if !ok {
		jww.ERROR.Printf("UpdateSentStatus msgID not found: %v",
			messageID)
		return
	}
	msg.status = status
}

func (r *receiver) DeleteMessage(message.ID, ed25519.PublicKey) bool {
	return true
}

func (r *receiver) GetConversation(ed25519.PublicKey) *dm.ModelConversation {
	return nil
}

func (r *receiver) GetConversations() []dm.ModelConversation {
	return nil
}