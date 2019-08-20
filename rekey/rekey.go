package rekey

import (
	"bytes"
	"fmt"
	"gitlab.com/elixxir/client/cmixproto"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/io"
	"gitlab.com/elixxir/client/keyStore"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/client/user"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/diffieHellman"
	"gitlab.com/elixxir/crypto/e2e"
	"gitlab.com/elixxir/crypto/hash"
	"gitlab.com/elixxir/primitives/circuit"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/switchboard"
)

var session user.Session
var topology *circuit.Circuit
var messaging io.Communications

var rekeyTriggerList rekeyTriggerListener
var rekeyList rekeyListener
var rekeyConfirmList rekeyConfirmListener

type rekeyTriggerListener struct {
	err error
}

func (l *rekeyTriggerListener) Hear(msg switchboard.Item, isHeardElsewhere bool) {
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

func (l *rekeyListener) Hear(msg switchboard.Item, isHeardElsewhere bool) {
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

func (l *rekeyConfirmListener) Hear(msg switchboard.Item, isHeardElsewhere bool) {
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
func InitRekey(s user.Session, m io.Communications, t *circuit.Circuit) {

	rekeyTriggerList = rekeyTriggerListener{}
	rekeyList = rekeyListener{}
	rekeyConfirmList = rekeyConfirmListener{}

	session = s
	topology = t
	messaging = m
	l := session.GetSwitchboard()

	l.Register(s.GetCurrentUser().User,
		int32(cmixproto.Type_REKEY_TRIGGER),
		&rekeyTriggerList)
	// TODO(nen) Wouldn't it be possible to register these listeners based
	//  solely on the inner type? maybe the switchboard can rebroadcast
	//  messages that have a type that includes the outer type if that's not
	//  possible
	// in short, switchboard should be the package that includes outer
	l.Register(id.ZeroID,
		int32(cmixproto.Type_NO_TYPE),
		&rekeyList)
	l.Register(id.ZeroID,
		int32(cmixproto.Type_REKEY_CONFIRM),
		&rekeyConfirmList)
}

type rekeyType uint8

const (
	None rekeyType = iota
	RekeyTrigger
	Rekey
	RekeyConfirm
)

func rekeyProcess(rt rekeyType, partner *id.User, data []byte) error {
	rkm := session.GetRekeyManager()
	grp := session.GetCmixGroup()
	e2egrp := session.GetE2EGroup()

	globals.Log.INFO.Printf("grp fingerprint: %d, e2e fingerprint: %d",
		grp.GetFingerprint(), e2egrp.GetFingerprint())

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
			privKeyCyclic = grp.RandomCoprime(grp.NewInt(1))
			fmt.Println("Private key actual: ", privKeyCyclic.Text(16))
			pubKeyCyclic = grp.ExpG(privKeyCyclic, grp.NewInt(1))
			// Get Current Partner Public Key from RekeyKeys
			partnerPubKeyCyclic = keys.CurrPubKey
			// Set new Own Private Key
			keys.NewPrivKey = privKeyCyclic
		} else {
			// Get Current Own Private Key from RekeyKeys
			privKeyCyclic = keys.CurrPrivKey
			// Get Partner New Public Key from data
			partnerPubKeyCyclic = grp.NewIntFromBytes(data)
			// Set new Partner Public Key
			keys.NewPubKey = partnerPubKeyCyclic
		}

		// Generate baseKey
		baseKey, _ = diffieHellman.CreateDHSessionKey(
			partnerPubKeyCyclic,
			privKeyCyclic,
			grp)

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
		// Delete current receive KeyManager
		oldKm := session.GetKeyStore().GetRecvManager(partner)
		oldKm.Destroy(session.GetKeyStore())
		// Create Receive KeyManager
		km := keyStore.NewManager(ctx.BaseKey, ctx.PrivKey, ctx.PubKey,
			partner, false,
			numKeys, keysTTL, params.NumRekeys)
		// Generate Receive Keys
		km.GenerateKeys(e2egrp, session.GetCurrentUser().User, session.GetKeyStore())
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
			km.GenerateKeys(e2egrp, session.GetCurrentUser().User, session.GetKeyStore())
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
		return messaging.SendMessageNoPartition(session, topology, partner, parse.E2E,
			pubKeyCyclic.LeftpadBytes(uint64(format.ContentsLen)))
	case Rekey:
		// Send rekey confirm message with hash of the baseKey
		h, _ := hash.NewCMixHash()
		h.Write(ctx.BaseKey.Bytes())
		baseKeyHash := h.Sum(nil)
		msg := parse.Pack(&parse.TypedBody{
			MessageType: int32(cmixproto.Type_REKEY_CONFIRM),
			Body:        baseKeyHash,
		})
		return messaging.SendMessage(session, topology, partner, parse.None, msg)
	}
	return nil
}
