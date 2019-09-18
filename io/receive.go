package io

import (
	"crypto/rand"
	"fmt"
	"gitlab.com/elixxir/client/cmixproto"
	"gitlab.com/elixxir/client/crypto"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/client/user"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/csprng"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/switchboard"
	"math/big"
	"time"
)

// MessageReceiver is a polling thread for receiving messages
func (cm *CommManager) MessageReceiver(session user.Session, delay time.Duration) {
	// FIXME: It's not clear we should be doing decryption here.
	if session == nil {
		globals.Log.FATAL.Panicf("No user session available")
	}
	pollingMessage := pb.ClientRequest{
		UserID: session.GetCurrentUser().User.Bytes(),
	}

	receiveGateway := id.NewNodeFromBytes(cm.ndf.Nodes[cm.ReceptionGatewayIndex].ID).NewGateway()

	quit := session.GetQuitChan()

	for {
		timerDelay := time.NewTimer(delay)
		select {
		case <-quit:
			globals.Log.DEBUG.Printf("Stopped message receiver\n")
			return
		case <-timerDelay.C:
			globals.Log.DEBUG.Printf("Attempting to receive message from gateway")
			decryptedMessages, senders, err := cm.receiveMessagesFromGateway(session, &pollingMessage, receiveGateway)
			if err != nil {

				backoffCount := 0

				for notConnected := true; notConnected; {

					cm.Disconnect()

					block, backoffTime := cm.computeBackoff(backoffCount)

					cm.setConnectionStatus(Offline, toSeconds(backoffTime))

					timer := time.NewTimer(backoffTime)

					if block {
						timer.Stop()
					}

					select {
					case <-session.GetQuitChan():
						close(session.GetQuitChan())
						return
					case <-timer.C:
					case <-cm.tryReconnect:
						backoffCount = 0
					}

					//call the callback with the connecting status

					err := cm.ConnectToGateways()

					if err == nil {
						notConnected = false
					}

					backoffCount++
				}
				//call the callback with the connected status
			}
			if decryptedMessages != nil {
				for i := range decryptedMessages {
					// TODO Handle messages that do not need partitioning
					assembledMessage := cm.collator.AddMessage(decryptedMessages[i],
						senders[i], time.Minute)
					if assembledMessage != nil {
						// we got a fully assembled message. let's broadcast it
						broadcastMessageReception(assembledMessage, session.GetSwitchboard())
					}
				}
			}
		}
	}
}

func (cm *CommManager) TryReconnect() {
	select {
	case cm.tryReconnect <- struct{}{}:
	default:
	}
}

func (cm *CommManager) computeBackoff(count int) (bool, time.Duration) {
	if count < maxAttempts {
		return true, 100 * time.Hour
	}

	wait := 2 ^ count
	if wait > maxBackoffTime {
		wait = maxBackoffTime
	}

	jitter, _ := rand.Int(csprng.NewSystemRNG(), big.NewInt(1000))
	backoffTime := time.Second*time.Duration(wait) + time.Millisecond*time.Duration(jitter.Int64())

	return false, backoffTime
}

func handleE2EReceiving(session user.Session,
	message *format.Message) (*id.User, bool, error) {
	keyFingerprint := message.GetKeyFP()

	// Lookup reception key
	recpKey := session.GetKeyStore().
		GetRecvKey(keyFingerprint)

	rekey := false
	if recpKey == nil {
		// TODO Handle sending error message to SW
		return nil, false, fmt.Errorf("E2EKey for matching fingerprint not found, can't process message")
	} else if recpKey.GetOuterType() == parse.Rekey {
		// If key type is rekey, the message is a rekey from partner
		rekey = true
	}

	sender := recpKey.GetManager().GetPartner()

	globals.Log.DEBUG.Printf("E2E decrypting message")
	var err error
	if rekey {
		err = crypto.E2EDecryptUnsafe(session.GetE2EGroup(), recpKey.GetKey(), message)
	} else {
		err = crypto.E2EDecrypt(session.GetE2EGroup(), recpKey.GetKey(), message)
	}

	if err != nil {
		// TODO handle Garbled message to SW
	}

	// Get partner from Key Manager of receiving key
	// since there is no space in message for senderID
	// Get decrypted partner public key from message
	// Send rekey message to switchboard
	if rekey {
		partner := recpKey.GetManager().GetPartner()
		partnerPubKey := message.Contents.Get()
		rekeyMsg := &parse.Message{
			Sender: partner,
			TypedBody: parse.TypedBody{
				MessageType: int32(cmixproto.Type_NO_TYPE),
				Body:        partnerPubKey,
			},
			InferredType: parse.Rekey,
			Receiver:     session.GetCurrentUser().User,
		}
		go session.GetSwitchboard().Speak(rekeyMsg)
	}
	return sender, rekey, err
}

func (cm *CommManager) receiveMessagesFromGateway(session user.Session,
	pollingMessage *pb.ClientRequest, receiveGateway *id.Gateway) ([]*format.Message, []*id.User, error) {
	if session != nil {
		pollingMessage.LastMessageID = session.GetLastMessageID()
		//FIXME: dont do this over an over

		messageIDs, err := cm.Comms.SendCheckMessages(receiveGateway,
			pollingMessage)

		if err != nil {
			globals.Log.WARN.Printf("CheckMessages error during polling: %v", err.Error())
			return nil, nil, err
		}

		globals.Log.DEBUG.Printf("Checking novelty of %v messageIDs", len(messageIDs.IDs))

		messages := make([]*format.Message, 0, len(messageIDs.IDs))
		senders := make([]*id.User, 0, len(messageIDs.IDs))
		for _, messageID := range messageIDs.IDs {
			// Get the first unseen message from the list of IDs
			_, received := cm.ReceivedMessages[messageID]
			if !received {
				globals.Log.INFO.Printf("Got a message waiting on the gateway: %v",
					messageID)
				// We haven't seen this message before.
				// So, we should retrieve it from the gateway.
				newMessage, err := cm.Comms.SendGetMessage(
					receiveGateway,
					&pb.ClientRequest{
						UserID: session.GetCurrentUser().User.
							Bytes(),
						LastMessageID: messageID,
					})
				if err != nil {
					globals.Log.WARN.Printf(
						"Couldn't receive message with ID %v while"+
							" polling gateway", messageID)
				} else {
					if newMessage.PayloadA == nil ||
						newMessage.PayloadB == nil {
						globals.Log.INFO.Println("Message fields not populated")
						continue
					}

					msg := format.NewMessage()
					msg.SetPayloadA(newMessage.PayloadA)
					msg.SetDecryptedPayloadB(newMessage.PayloadB)
					var err error = nil
					var rekey bool
					var unpadded []byte
					var sender *id.User
					// If message is E2E, handle decryption
					if !e2e.IsUnencrypted(msg) {
						sender, rekey, err = handleE2EReceiving(session, msg)
					} else {
						// If message is non E2E, need to unpad payload
						unpadded, err = e2e.Unpad(msg.Contents.Get())
						if err == nil {
							msg.Contents.SetRightAligned(unpadded)
						}
						sender = id.NewUserFromBytes(msg.GetMAC())
					}

					if err != nil {
						globals.Log.WARN.Printf(
							"Message did not decrypt properly, "+
								"not adding to messages array: %v", err.Error())
					} else if rekey {
						globals.Log.INFO.Printf("Correctly processed rekey message," +
							" not adding to messages array")
					} else {
						messages = append(messages, msg)
						senders = append(senders, sender)
					}

					globals.Log.INFO.Printf(
						"Adding message ID %v to received message IDs", messageID)
					cm.ReceivedMessages[messageID] = struct{}{}
					session.SetLastMessageID(messageID)
					err = session.StoreSession()
					if err != nil {
						globals.Log.ERROR.Printf("Could not store session "+
							"after message received from gateway: %+v", err)
					}
				}
			}
		}
		return messages, senders, nil
	}
	return nil, nil, nil
}

func broadcastMessageReception(message *parse.Message,
	listeners *switchboard.Switchboard) {

	listeners.Speak(message)
}
