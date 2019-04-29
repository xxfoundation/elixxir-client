package rekey

import (
	"bytes"
	"crypto/rand"
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
	"gitlab.com/elixxir/crypto/signature"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/switchboard"
)

var session user.Session
var messaging io.Communications

var rekeyTriggerList rekeyTriggerListener
var rekeyList rekeyListener
var rekeyConfirmList rekeyConfirmListener

type rekeyTriggerListener struct{
	err error
}

func (l *rekeyTriggerListener) Hear(msg switchboard.Item, isHeardElsewhere bool) {
	m := msg.(*parse.Message)
	partner := m.GetRecipient()
	pubKey := m.GetPayload()
	err := rekeyProcess(RekeyTrigger, partner, pubKey)
	if err != nil {
		globals.Log.WARN.Printf("Error on rekeyProcess: %s", err.Error())
		l.err = err
	} else {
		l.err = nil
	}
}

type rekeyListener struct{
	err error
}

func (l *rekeyListener) Hear(msg switchboard.Item, isHeardElsewhere bool) {
	m := msg.(*parse.Message)
	partner := m.GetSender()
	keys := m.GetPayload()
	err := rekeyProcess(Rekey, partner, keys)
	if err != nil {
		globals.Log.WARN.Printf("Error on rekeyProcess: %s", err.Error())
		l.err = err
	} else {
		l.err = nil
	}
}

type rekeyConfirmListener struct{
	err error
}

func (l *rekeyConfirmListener) Hear(msg switchboard.Item, isHeardElsewhere bool) {
	m := msg.(*parse.Message)
	partner := m.GetSender()
	baseKeyHash := m.GetPayload()
	err := rekeyProcess(RekeyConfirm, partner, baseKeyHash)
	if err != nil {
		globals.Log.WARN.Printf("Error on rekeyProcess: %s", err.Error())
		l.err = err
	} else {
		l.err = nil
	}
}

// InitRekey is called internally by the Login API
func InitRekey(s user.Session, m io.Communications) {

	rekeyTriggerList = rekeyTriggerListener{}
	rekeyList = rekeyListener{}
	rekeyConfirmList = rekeyConfirmListener{}

	session = s
	messaging = m
	l := session.GetSwitchboard()

	l.Register(s.GetCurrentUser().User,
		format.None, int32(cmixproto.Type_REKEY_TRIGGER),
		&rekeyTriggerList)
	l.Register(id.ZeroID,
		format.Rekey, int32(cmixproto.Type_NO_TYPE),
		&rekeyList)
	l.Register(id.ZeroID,
		format.None, int32(cmixproto.Type_REKEY_CONFIRM),
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
	grp := session.GetGroup()

	// Error handling according to Rekey Message Type
	var ctx *keyStore.RekeyContext
	var privKey []byte
	var partnerPubKey []byte
	switch rt {
	case RekeyTrigger:
		ctx = rkm.GetCtx(partner)
		if ctx != nil {
			return fmt.Errorf("rekey already in progress with user %v,"+
				" ignoring repetition", *partner)
		}
		// Get partner PublicKey from data
		partnerPubKey = data
	case Rekey:
		// Get private Key and public Key from data
		privKey = data[:format.TOTAL_LEN]
		partnerPubKey = data[format.TOTAL_LEN:]
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
	var baseKey *cyclic.Int
	var pubKey *signature.DSAPublicKey
	if ctx == nil {
		if privKey == nil {
			params := signature.GetDefaultDSAParams()
			privateKey := params.PrivateKeyGen(rand.Reader)
			pubKey = privateKey.PublicKeyGen()
			privKeyCyclic = grp.NewIntFromLargeInt(privateKey.GetKey())
		} else {
			privKeyCyclic = grp.NewIntFromBytes(privKey)
		}

		pubKeyCyclic = grp.NewIntFromBytes(partnerPubKey)
		// Generate baseKey
		baseKey, _ = diffieHellman.CreateDHSessionKey(
			pubKeyCyclic,
			privKeyCyclic,
			grp)
		ctx = &keyStore.RekeyContext{
			BaseKey: baseKey,
			PrivKey: privKeyCyclic,
			PubKey:  pubKeyCyclic,
		}

		if rt == RekeyTrigger {
			rkm.AddCtx(partner, ctx)
		}
	}

	// Generate key TTL and number of keys
	keysTTL, numKeys := e2e.GenerateKeyTTL(ctx.BaseKey.GetLargeInt(),
		keyStore.MinKeys, keyStore.MaxKeys,
		e2e.TTLParams{keyStore.TTLScalar,
			keyStore.Threshold})
	// Create Key Manager if needed
	switch rt {
	case Rekey:
		// Delete current receive KeyManager
		oldKm := session.GetKeyStore().GetRecvManager(partner)
		oldKm.Destroy(session.GetKeyStore())
		// Create Receive KeyManager
		km := keyStore.NewManager(ctx.BaseKey, ctx.PrivKey, ctx.PubKey,
			partner, false,
			numKeys, keysTTL, keyStore.NumReKeys)
		// Generate Receive Keys
		km.GenerateKeys(grp, session.GetCurrentUser().User, session.GetKeyStore())
		// Remove RekeyContext
		rkm.DeleteCtx(partner)
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
				numKeys, keysTTL, keyStore.NumReKeys)
			// Generate Send Keys
			km.GenerateKeys(grp, session.GetCurrentUser().User, session.GetKeyStore())
			// Remove RekeyContext
			rkm.DeleteCtx(partner)
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
		return messaging.SendMessageNoPartition(session, partner, format.E2E,
			pubKey.GetKey().LeftpadBytes(uint64(format.TOTAL_LEN)))
	case Rekey:
		// Send rekey confirm message with hash of the baseKey
		h, _ := hash.NewCMixHash()
		h.Write(ctx.BaseKey.Bytes())
		msg := parse.Pack(&parse.TypedBody{
			MessageType: int32(cmixproto.Type_REKEY_CONFIRM),
			Body: h.Sum(nil),
		})
		return messaging.SendMessage(session, partner, format.None, msg)
	}
	return nil
}
