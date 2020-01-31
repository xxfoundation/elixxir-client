package user

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/keyStore"
	"gitlab.com/elixxir/crypto/cyclic"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/primitives/format"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/switchboard"
	"sync"
)

// Struct holding relevant session data
type SessionObjV1 struct {
	// Currently authenticated user
	CurrentUser *UserV1

	Keys             map[id.Node]NodeKeys
	RSAPrivateKey    *rsa.PrivateKey
	RSAPublicKey     *rsa.PublicKey
	CMIXDHPrivateKey *cyclic.Int
	CMIXDHPublicKey  *cyclic.Int
	E2EDHPrivateKey  *cyclic.Int
	E2EDHPublicKey   *cyclic.Int
	CmixGrp          *cyclic.Group
	E2EGrp           *cyclic.Group
	Salt             []byte

	// Last received message ID. Check messages after this on the gateway.
	LastMessageID string

	//Interface map for random data storage
	InterfaceMap map[string]interface{}

	// E2E KeyStore
	KeyMaps *keyStore.KeyStore

	// Rekey Manager
	RekeyManager *keyStore.RekeyManager

	// Non exported fields (not GOB encoded/decoded)
	// Local pointer to storage of this session
	store globals.Storage

	// Switchboard
	listeners *switchboard.Switchboard

	// Quit channel for message reception runner
	quitReceptionRunner chan struct{}

	lock sync.Mutex

	// The password used to encrypt this session when saved
	password string

	//The validation signature provided by permissioning
	regValidationSignature []byte

	// Buffer of messages that cannot be decrypted
	garbledMessages []*format.Message

	RegState *uint32

	storageLocation uint8

	ContactsByValue map[string]SearchedUserRecord
}

// Struct representing a User in the system
type UserV1 struct {
	User  *id.User
	Nick  string
	Email string
}

// ConvertSessionV1toV2 converts the session object from version 1 to version 2.
// This conversion includes:
//  1. Changing the RegState values to the new integer values (1 to 2000, and 2
//     to 3000).
func ConvertSessionV1toV2(inputWrappedSession *SessionStorageWrapper) (*SessionStorageWrapper, error) {
	//extract teh session from the wrapper
	var sessionBytes bytes.Buffer

	//get the old session object
	sessionBytes.Write(inputWrappedSession.Session)
	dec := gob.NewDecoder(&sessionBytes)

	sessionV1 := SessionObjV1{}

	err := dec.Decode(&sessionV1)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to decode session")
	}

	sessionV2 := SessionObj{}

	// Convert RegState to new values
	if *sessionV1.RegState == 1 {
		*sessionV1.RegState = 2000
	} else if *sessionV1.RegState == 2 {
		*sessionV1.RegState = 3000
	}

	//convert the user object
	sessionV2.CurrentUser = &User{
		User:     sessionV1.CurrentUser.User,
		Username: sessionV1.CurrentUser.Email,
	}

	//port identical values over
	sessionV2.NodeKeys = sessionV1.Keys
	sessionV2.RSAPrivateKey = sessionV1.RSAPrivateKey
	sessionV2.CMIXDHPrivateKey = sessionV1.CMIXDHPrivateKey
	sessionV2.CMIXDHPublicKey = sessionV1.CMIXDHPublicKey
	sessionV2.E2EDHPrivateKey = sessionV1.E2EDHPrivateKey
	sessionV2.E2EDHPublicKey = sessionV1.E2EDHPublicKey
	sessionV2.CmixGrp = sessionV1.CmixGrp
	sessionV2.E2EGrp = sessionV1.E2EGrp
	sessionV2.Salt = sessionV1.Salt
	sessionV2.LastMessageID = sessionV1.LastMessageID
	sessionV2.InterfaceMap = sessionV1.InterfaceMap
	sessionV2.KeyMaps = sessionV1.KeyMaps
	sessionV2.RekeyManager = sessionV1.RekeyManager
	sessionV2.RegValidationSignature = sessionV1.regValidationSignature
	sessionV2.RegState = sessionV1.RegState
	sessionV2.ContactsByValue = sessionV1.ContactsByValue

	//re encode the session
	var sessionBuffer bytes.Buffer

	enc := gob.NewEncoder(&sessionBuffer)

	err = enc.Encode(sessionV2)

	if err != nil {
		err = errors.New(fmt.Sprintf("ConvertSessionV1toV2: Could not "+
			" store session v2: %s", err.Error()))
		return nil, err
	}

	//build the session wrapper
	ssw := SessionStorageWrapper{
		Version:   2,
		Timestamp: inputWrappedSession.Timestamp,
		Session:   sessionBuffer.Bytes(),
	}

	return &ssw, nil
}
