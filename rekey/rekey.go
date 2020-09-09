package rekey

import (
	"bytes"
	"fmt"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/keyStore"
	"gitlab.com/elixxir/client/network"
	"gitlab.com/elixxir/client/network/keyExchange"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/user"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/switchboard"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
)

var session user.Session
var sessionV2 storage.Session
var topology *connect.Circuit
var comms network.Communications
var transmissionHost *connect.Host

var rekeyTriggerList rekeyTriggerListener
var rekeyList rekeyListener
var rekeyConfirmList rekeyConfirmListener

var rekeyChan chan struct{}

type rekeyTriggerListener struct {
	err error
}

func (l *rekeyTriggerListener) Hear(msg switchboard.Item, isHeardElsewhere bool, i ...interface{}) {
	m := msg.(*parse.Message)
	partner := m.GetRecipient()
	globals.Log.DEBUG.Printf("Received RekeyTrigger message for user %v", *partner)
	err := rekeyProcess(RekeyTrigger, partner, nil)
	if err != nil {
		globals.Log.WARN.Printf("Error on rekeyProcess: %s", err.Error())
		l.err = err
	} else {
		l.err = nil
	}
}

type rekeyListener struct {
	err error
}

func (l *rekeyListener) Hear(msg switchboard.Item, isHeardElsewhere bool, i ...interface{}) {
	m := msg.(*parse.Message)
	partner := m.GetSender()
	partnerPubKey := m.GetPayload()
	if m.GetCryptoType() != parse.Rekey {
		globals.Log.WARN.Printf("Received message with NO_TYPE but not Rekey CryptoType, needs to be fixed!")
		return
	}
	globals.Log.DEBUG.Printf("Received Rekey message from user %v", *partner)
	err := rekeyProcess(Rekey, partner, partnerPubKey)
	if err != nil {
		globals.Log.WARN.Printf("Error on rekeyProcess: %s", err.Error())
		l.err = err
	} else {
		l.err = nil
	}
}

type rekeyConfirmListener struct {
	err error
}

func (l *rekeyConfirmListener) Hear(msg switchboard.Item, isHeardElsewhere bool, i ...interface{}) {
	m := msg.(*parse.Message)
	partner := m.GetSender()
	baseKeyHash := m.GetPayload()
	globals.Log.DEBUG.Printf("Received RekeyConfirm message from user %v", *partner)
	err := rekeyProcess(RekeyConfirm, partner, baseKeyHash)
	if err != nil {
		globals.Log.WARN.Printf("Error on rekeyProcess: %s", err.Error())
		l.err = err
	} else {
		l.err = nil
	}
}

// InitRekey is called internally by the Login API
func InitRekey(s user.Session, s2 storage.Session, m network.Communications,
	t *connect.Circuit, host *connect.Host, rekeyChan2 chan struct{}) {

	rekeyTriggerList = rekeyTriggerListener{}
	rekeyList = rekeyListener{}
	rekeyConfirmList = rekeyConfirmListener{}

	session = s
	sessionV2 = s2
	topology = t
	comms = m
	transmissionHost = host

	rekeyChan = rekeyChan2
	l := m.GetSwitchboard()

	userData, err := s2.GetUserData()
	if err != nil {
		globals.Log.FATAL.Panicf("could not load user data: %+v", err)
	}

	l.Register(userData.ThisUser.User,
		int32(keyExchange.Type_REKEY_TRIGGER),
		&rekeyTriggerList)
	// TODO(nen) Wouldn't it be possible to register these listeners based
	//  solely on the inner type? maybe the switchboard can rebroadcast
	//  messages that have a type that includes the outer type if that's not
	//  possible
	// in short, switchboard should be the package that includes outer
	l.Register(&id.ZeroUser,
		int32(keyExchange.Type_NO_TYPE),
		&rekeyList)
	l.Register(&id.ZeroUser,
		int32(keyExchange.Type_REKEY_CONFIRM),
		&rekeyConfirmList)
}

type rekeyType uint8

const (
	None rekeyType = iota
	RekeyTrigger
	Rekey
	RekeyConfirm
)

func rekeyProcess(rt rekeyType, partner *id.ID, data []byte) error {
	rkm := session.GetRekeyManager()
	userData, err := sessionV2.GetUserData()
	if err != nil {
		return fmt.Errorf("could not load user data: %+v", err)
	}

	e2egrp := userData.E2EGrp

	// Error handling according to Rekey Message Type
	var ctx *keyStore.RekeyContext
	var keys *keyStore.RekeyKeys
	switch rt {
	case RekeyTrigger:
		ctx = rkm.GetCtx(partner)
		if ctx != nil {
			return fmt.Errorf("rekey already in progress with user %v,"+
				" ignoring repetition", *partner)
		}
		keys = rkm.GetKeys(partner)
		if keys == nil {
			return fmt.Errorf("couldn't get RekeyKeys object for user: %v", *partner)
		}
	case Rekey:
		keys = rkm.GetKeys(partner)
		if keys == nil {
			return fmt.Errorf("couldn't get RekeyKeys object for user: %v", *partner)
		}
	case RekeyConfirm:
		ctx = rkm.GetCtx(partner)
		if ctx == nil {
			return fmt.Errorf("rekey not in progress with user %v,"+
				" ignoring confirmation", *partner)
		}
	}

	// Create Rekey Context if not existing
	// Use set privKey and partner pubKey for Rekey
	// For RekeyTrigger, generate new privKey / pubKey pair
	// Add context to RekeyManager in case of RekeyTrigger
	var privKeyCyclic *cyclic.Int
	var pubKeyCyclic *cyclic.Int
	var partnerPubKeyCyclic *cyclic.Int
	var baseKey *cyclic.Int
	if ctx == nil {
		if rt == RekeyTrigger {
			privKeyCyclic = e2egrp.RandomCoprime(e2egrp.NewInt(1))
			globals.Log.DEBUG.Println("Private key actual: ", privKeyCyclic.Text(16))
			pubKeyCyclic = e2egrp.ExpG(privKeyCyclic, e2egrp.NewInt(1))
			// Get Current Partner Public Key from RekeyKeys
			partnerPubKeyCyclic = keys.CurrPubKey
			// Set new Own Private Key
			keys.NewPrivKey = privKeyCyclic
		} else {
			// Get Current Own Private Key from RekeyKeys
			privKeyCyclic = keys.CurrPrivKey
			// Get Partner New Public Key from data
			partnerPubKeyCyclic = e2egrp.NewIntFromBytes(data)
			// Set new Partner Public Key
			keys.NewPubKey = partnerPubKeyCyclic
		}

		// Generate baseKey
		baseKey, _ = diffieHellman.CreateDHSessionKey(
			partnerPubKeyCyclic,
			privKeyCyclic,
			e2egrp)

		ctx = &keyStore.RekeyContext{
			BaseKey: baseKey,
			PrivKey: privKeyCyclic,
			PubKey:  partnerPubKeyCyclic,
		}

		if rt == RekeyTrigger {
			rkm.AddCtx(partner, ctx)
		}
		// Rotate Keys if ready
		keys.RotateKeysIfReady()
	}

	// Generate key TTL and number of keys
	params := session.GetKeyStore().GetKeyParams()
	keysTTL, numKeys := e2e.GenerateKeyTTL(ctx.BaseKey.GetLargeInt(),
		params.MinKeys, params.MaxKeys, params.TTLParams)
	// Create Key Manager if needed
	switch rt {
	case Rekey:
		// Create Receive KeyManager
		km := keyStore.NewManager(ctx.BaseKey, ctx.PrivKey, ctx.PubKey,
			partner, false,
			numKeys, keysTTL, params.NumRekeys)
		// Generate Receive Keys
		e2ekeys := km.GenerateKeys(e2egrp, userData.ThisUser.User)
		session.GetKeyStore().AddRecvManager(km)
		session.GetKeyStore().AddReceiveKeysByFingerprint(e2ekeys)

		globals.Log.DEBUG.Printf("Generated new receiving keys for E2E"+
			" relationship with user %v", *partner)
	case RekeyConfirm:
		// Check baseKey Hash matches expected
		h, _ := hash.NewCMixHash()
		h.Write(ctx.BaseKey.Bytes())
		expected := h.Sum(nil)
		if bytes.Equal(expected, data) {
			// Delete current send KeyManager
			oldKm := session.GetKeyStore().GetSendManager(partner)
			oldKm.Destroy(session.GetKeyStore())
			// Create Send KeyManager
			km := keyStore.NewManager(ctx.BaseKey, ctx.PrivKey, ctx.PubKey,
				partner, true,
				numKeys, keysTTL, params.NumRekeys)
			// Generate Send Keys
			km.GenerateKeys(e2egrp, userData.ThisUser.User)
			session.GetKeyStore().AddSendManager(km)
			// Remove RekeyContext
			rkm.DeleteCtx(partner)
			globals.Log.DEBUG.Printf("Generated new send keys for E2E"+
				" relationship with user %v", *partner)
		} else {
			return fmt.Errorf("rekey-confirm from user %v failed,"+
				" baseKey hash doesn't match expected", *partner)
		}
	}

	// Send message if needed
	switch rt {
	case RekeyTrigger:
		// Directly send raw publicKey bytes, without any message type
		// This ensures that the publicKey fits in a single message, which
		// is sent with E2E encryption using a send Rekey, and without padding
		return comms.SendMessageNoPartition(session, topology, partner, parse.E2E,
			pubKeyCyclic.LeftpadBytes(uint64(format.ContentsLen)), transmissionHost)
	case Rekey:
		// Trigger the rekey channel
		select {
		case rekeyChan <- struct{}{}:
		}

		// Send rekey confirm message with hash of the baseKey
		h, _ := hash.NewCMixHash()
		h.Write(ctx.BaseKey.Bytes())
		baseKeyHash := h.Sum(nil)
		msg := parse.Pack(&parse.TypedBody{
			MessageType: int32(keyExchange.Type_REKEY_CONFIRM),
			Body:        baseKeyHash,
		})
		return comms.SendMessage(session, topology, partner, parse.None, msg, transmissionHost)
	}
	return nil
}
