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

type rekeyTriggerListener struct{}

func (l *rekeyTriggerListener) Hear(msg switchboard.Item, isHeardElsewhere bool) {
	m := msg.(*parse.Message)
	partner := m.GetRecipient()
	partnerPubKey := m.GetPayload()
	err := rekeyProcess(RekeyTrigger, partner,
		nil, partnerPubKey, nil)
	if err != nil {
		globals.Log.WARN.Printf("Error on rekeyProcess: %s", err.Error())
	}
}

type rekeyListener struct{}

func (l *rekeyListener) Hear(msg switchboard.Item, isHeardElsewhere bool) {
	m := msg.(*parse.Message)
	partner := m.GetSender()
	keys := m.GetPayload()
	privKey := keys[:format.TOTAL_LEN]
	partnerPubKey := keys[format.TOTAL_LEN:]
	err := rekeyProcess(Rekey, partner,
		privKey, partnerPubKey, nil)
	if err != nil {
		globals.Log.WARN.Printf("Error on rekeyProcess: %s", err.Error())
	}
}

type rekeyConfirmListener struct{}

func (l *rekeyConfirmListener) Hear(msg switchboard.Item, isHeardElsewhere bool) {
	m := msg.(*parse.Message)
	partner := m.GetSender()
	baseKeyHash := m.GetPayload()
	err := rekeyProcess(RekeyConfirm, partner,
		nil, nil, baseKeyHash)
	if err != nil {
		globals.Log.WARN.Printf("Error on rekeyProcess: %s", err.Error())
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

func rekeyProcess(rt rekeyType,
	partner *id.User, privKey,
	partnerPubKey, baseKeyHash []byte) error {
	rkm := session.GetRekeyManager()
	grp := session.GetGroup()

	// Error handling according to Rekey Message Type
	var ctx *keyStore.RekeyContext
	switch rt {
	case RekeyTrigger:
		ctx = rkm.GetOutCtx(partner)
		if ctx != nil {
			return fmt.Errorf("rekey already in progress with user %v,"+
				" ignoring repetition", *partner)
		}
	case Rekey:
		ctx = rkm.GetInCtx(partner)
		if ctx != nil {
			return fmt.Errorf("rekey from user %v already in progress,"+
				" ignoring repetition", *partner)
		}
	case RekeyConfirm:
		ctx = rkm.GetOutCtx(partner)
		if ctx == nil {
			return fmt.Errorf("rekey not in progress with user %v,"+
				" ignoring confirmation", *partner)
		}
	}

	// Create Rekey Context if not existing
	// Use set privKey and pubKey for Rekey
	// For RekeyTrigger, generate new privKey / pubKey pair
	// Add context to correct Rekey Manager Map
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
			privKeyCyclic,
			pubKeyCyclic,
			grp)
		ctx = &keyStore.RekeyContext{
			BaseKey: baseKey,
			PrivKey: privKeyCyclic,
			PubKey:  pubKeyCyclic,
		}
		switch rt {
		case RekeyTrigger:
			rkm.AddOutCtx(partner, ctx)
		case Rekey:
			rkm.AddInCtx(partner, ctx)
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
		oldKm := session.GetRecvKeyManager(partner)
		oldKm.Destroy(session.GetKeyStore())
		session.DeleteRecvKeyManager(partner)
		// Create Receive KeyManager
		km := keyStore.NewManager(ctx.BaseKey, ctx.PrivKey, ctx.PubKey,
			partner, false,
			numKeys, keysTTL, keyStore.NumReKeys)
		// Generate Receive Keys
		km.GenerateKeys(grp, session.GetCurrentUser().User, session.GetKeyStore())
		// Add Receive Key Manager to session
		session.AddRecvKeyManager(km)
	case RekeyConfirm:
		// Check baseKey Hash matches expected
		h, _ := hash.NewCMixHash()
		h.Write(ctx.BaseKey.Bytes())
		expected := h.Sum(nil)
		if bytes.Equal(expected, baseKeyHash) {
			// Delete current send KeyManager
			oldKm := session.GetSendKeyManager(partner)
			oldKm.Destroy(session.GetKeyStore())
			session.DeleteSendKeyManager(partner)
			// Create Send KeyManager
			km := keyStore.NewManager(ctx.BaseKey, ctx.PrivKey, ctx.PubKey,
				partner, true,
				numKeys, keysTTL, keyStore.NumReKeys)
			// Generate Send Keys
			km.GenerateKeys(grp, session.GetCurrentUser().User, session.GetKeyStore())
			// Add Send Key Manager to session
			session.AddSendKeyManager(km)
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
