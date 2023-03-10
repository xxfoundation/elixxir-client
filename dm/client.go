////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package dm

import (
	"crypto/ed25519"
	"fmt"
	"strings"
	sync "sync"
	"time"

	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/codename"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/xx_network/primitives/id"

	"gitlab.com/elixxir/client/v4/cmix/identity"
	"gitlab.com/elixxir/client/v4/storage/versioned"
	"gitlab.com/elixxir/crypto/nike"
	"gitlab.com/elixxir/crypto/nike/ecdh"
)

const (
	nickStoreKey = "dm_nickname_%s"
)

type dmClient struct {
	me              *codename.PrivateIdentity
	selfReceptionID *id.ID
	receptionID     *id.ID
	privateKey      nike.PrivateKey
	publicKey       nike.PublicKey
	myToken         uint32
	receiver        EventModel

	st  SendTracker
	nm  NickNameManager
	net cMixClient
	rng *fastRNG.StreamGenerator
}

// NewDMClient creates a new client for direct messaging. This should
// be called when the channels manager is created/loaded. It has no
// associated state, so it does not have a corresponding Load
// function.
//
// The DMClient implements both the Sender and ListenerRegistrar interface.
// See send.go for implementation of the Sender interface.
func NewDMClient(myID *codename.PrivateIdentity, receiver EventModel,
	tracker SendTracker,
	nickManager NickNameManager,
	net cMixClient,
	rng *fastRNG.StreamGenerator) Client {

	privateEdwardsKey := myID.Privkey
	myIDToken := myID.GetDMToken()

	privateKey := ecdh.Edwards2ECDHNIKEPrivateKey(privateEdwardsKey)
	publicKey := ecdh.ECDHNIKE.DerivePublicKey(privateKey)

	receptionID := deriveReceptionID(publicKey.Bytes(), myIDToken)
	selfReceptionID := deriveReceptionID(privateKey.Bytes(), myIDToken)

	dmc := &dmClient{
		me:              myID,
		receptionID:     receptionID,
		selfReceptionID: selfReceptionID,
		privateKey:      privateKey,
		publicKey:       publicKey,
		myToken:         myIDToken,
		st:              tracker,
		nm:              nickManager,
		net:             net,
		rng:             rng,
		receiver:        receiver,
	}

	// Register the listener
	dmc.register(receiver, dmc.st)

	return dmc
}

// Register registers a listener for direct messages.
func (dc *dmClient) register(apiReceiver EventModel,
	tracker SendTracker) error {
	beginningOfTime := time.Time{}
	r := &receiver{
		c:           dc,
		api:         apiReceiver,
		sendTracker: tracker,
	}

	// Initialize Send Tracking
	dc.st.Init(dc.net, r.receiveMessage, r.api.UpdateSentStatus, dc.rng)

	// Start listening
	dc.net.AddIdentityWithHistory(dc.receptionID, identity.Forever,
		beginningOfTime, true, r.GetProcessor())

	dc.net.AddIdentityWithHistory(dc.selfReceptionID, identity.Forever,
		beginningOfTime, true, r.GetSelfProcessor())
	return nil
}

func NewNicknameManager(id *id.ID, ekv *versioned.KV) NickNameManager {
	return &nickMgr{
		ekv:      ekv,
		storeKey: fmt.Sprintf(nickStoreKey, id.String()),
		nick:     "",
	}
}

type nickMgr struct {
	storeKey string
	ekv      *versioned.KV
	nick     string
	sync.Mutex
}

func (dc *dmClient) GetPublicKey() nike.PublicKey {
	return dc.publicKey
}

func (dc *dmClient) GetToken() uint32 {
	return dc.myToken
}

// GetIdentity returns the public identity associated with this channel manager.
func (dc *dmClient) GetIdentity() codename.Identity {
	return dc.me.Identity
}

// GetNickname returns the stored nickname if there is one
func (dc *dmClient) GetNickname() (string, bool) {
	return dc.nm.GetNickname()
}

// SetNickname saves the nickname
func (dc *dmClient) SetNickname(nick string) {
	dc.nm.SetNickname(nick)
}

// IsBlocked returns if the given sender is blocked
// Blocking is controlled by the Receiver / EventModel
func (dc *dmClient) IsBlocked(senderPubKey ed25519.PublicKey) bool {
	conversation := dc.receiver.GetConversation(senderPubKey)
	if conversation != nil {
		return conversation.Blocked
	}
	return false
}

// GetBlockedSenders returns all senders who are blocked by this user.
// Blocking is controlled by the Receiver / EventModel
func (dc *dmClient) GetBlockedSenders() []ed25519.PublicKey {
	allConversations := dc.receiver.GetConversations()
	blocked := make([]ed25519.PublicKey, 0)
	for i := range allConversations {
		convo := allConversations[i]
		if convo.Blocked {
			pub := convo.Pubkey
			blocked = append(blocked, ed25519.PublicKey(pub))
		}
	}
	return blocked
}

// ExportPrivateIdentity encrypts and exports the private identity to a portable
// string.
func (dc *dmClient) ExportPrivateIdentity(password string) ([]byte, error) {
	jww.INFO.Printf("[DM] ExportPrivateIdentity()")
	rng := dc.rng.GetStream()
	defer rng.Close()
	return dc.me.Export(password, rng)
}

// GetNickname returns the stored nickname if there is one
func (nm *nickMgr) GetNickname() (string, bool) {
	nm.Lock()
	defer nm.Unlock()
	if nm.nick != "" {
		return nm.nick, true
	}
	nickObj, err := nm.ekv.Get(nm.storeKey, 0)
	if err != nil {
		return "", false
	}
	nm.nick = string(nickObj.Data)
	return nm.nick, true
}

// SetNickname saves the nickname
func (nm *nickMgr) SetNickname(nick string) {
	nm.Lock()
	defer nm.Unlock()
	nm.nick = strings.Clone(nick)
	nickObj := &versioned.Object{
		Version:   0,
		Timestamp: time.Now(),
		Data:      []byte(nick),
	}
	nm.ekv.Set(nm.storeKey, nickObj)
}
