///////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"crypto/ed25519"
	"encoding/json"
	clientNotif "gitlab.com/elixxir/client/v4/notifications"
	"sync"
	"time"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"

	"gitlab.com/elixxir/client/v4/channels"
	"gitlab.com/elixxir/client/v4/channels/storage"
	"gitlab.com/elixxir/client/v4/cmix/rounds"
	"gitlab.com/elixxir/client/v4/storage/utility"
	"gitlab.com/elixxir/client/v4/xxdk"
	cryptoBroadcast "gitlab.com/elixxir/crypto/broadcast"
	cryptoChannel "gitlab.com/elixxir/crypto/channel"
	"gitlab.com/elixxir/crypto/message"
	cryptoMessage "gitlab.com/elixxir/crypto/message"
	"gitlab.com/elixxir/primitives/notifications"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/id/ephemeral"
)

////////////////////////////////////////////////////////////////////////////////
// Singleton Tracker                                                          //
////////////////////////////////////////////////////////////////////////////////

// channelManagerTrackerSingleton is used to track ChannelsManager objects
// so that they can be referenced by ID back over the bindings.
var channelManagerTrackerSingleton = &channelManagerTracker{
	tracked: make(map[int]*ChannelsManager),
	count:   0,
}

// channelManagerTracker is a singleton used to keep track of extant
// ChannelsManager objects, preventing race conditions created by passing it
// over the bindings.
type channelManagerTracker struct {
	tracked map[int]*ChannelsManager
	count   int
	mux     sync.RWMutex
}

// make create a ChannelsManager from an [channels.Manager], assigns it a unique
// ID, and adds it to the channelManagerTracker.
func (cmt *channelManagerTracker) make(c channels.Manager) *ChannelsManager {
	cmt.mux.Lock()
	defer cmt.mux.Unlock()

	chID := cmt.count
	cmt.count++

	cmt.tracked[chID] = &ChannelsManager{
		api: c,
		id:  chID,
	}

	return cmt.tracked[chID]
}

// get an ChannelsManager from the channelManagerTracker given its ID.
func (cmt *channelManagerTracker) get(id int) (*ChannelsManager, error) {
	cmt.mux.RLock()
	defer cmt.mux.RUnlock()

	c, exist := cmt.tracked[id]
	if !exist {
		return nil, errors.Errorf(
			"Cannot get ChannelsManager for ID %d, does not exist", id)
	}

	return c, nil
}

// delete removes a ChannelsManager from the channelManagerTracker.
func (cmt *channelManagerTracker) delete(id int) {
	cmt.mux.Lock()
	defer cmt.mux.Unlock()

	delete(cmt.tracked, id)
}

// channelManagerTrackerSingleton is used to track channels.ExtensionBuilder
// objects so that they can be referenced by ID back over the bindings.
var channelExtensionBuilderTrackerSingleton = &channelExtensionBuilderTracker{
	tracked: make(map[int]channels.ExtensionBuilder),
	count:   0,
}

// channelExtensionBuilderTracker is a singleton used to keep track of extant
// channels.ExtensionBuilder objects, preventing race conditions created by
// passing it over the bindings.
type channelExtensionBuilderTracker struct {
	tracked map[int]channels.ExtensionBuilder
	count   int
	mux     sync.RWMutex
}

// make create a ChannelsManager from an [channels.Manager], assigns it a unique
// ID, and adds it to the channelManagerTracker.
func (ebt *channelExtensionBuilderTracker) add(eb channels.ExtensionBuilder) int {
	ebt.mux.Lock()
	defer ebt.mux.Unlock()

	ebID := ebt.count
	ebt.count++

	ebt.tracked[ebID] = eb

	return ebID
}

// get returns a list of channels.ExtensionBuilder associated with each ID in
// the list. If the ID does not exist, it is skipped. All invalid IDs results in
// an empty list.
func (ebt *channelExtensionBuilderTracker) get(
	ids ...int) []channels.ExtensionBuilder {
	ebt.mux.RLock()
	defer ebt.mux.RUnlock()

	list := make([]channels.ExtensionBuilder, 0, len(ids))
	for _, ebID := range ids {
		if eb, exists := ebt.tracked[ebID]; exists {
			list = append(list, eb)
		}
	}

	return list
}

// delete removes a channels.ExtensionBuilder from the tracker.
func (ebt *channelExtensionBuilderTracker) delete(ebID int) {
	ebt.mux.Lock()
	defer ebt.mux.Unlock()

	delete(ebt.tracked, ebID)
}

////////////////////////////////////////////////////////////////////////////////
// Basic Channel API                                                          //
////////////////////////////////////////////////////////////////////////////////

// ChannelsManager is a bindings-layer struct that wraps a [channels.Manager]
// interface.
type ChannelsManager struct {
	api channels.Manager
	id  int
}

// GetID returns the unique tracking ID for the [ChannelsManager] object.
func (cm *ChannelsManager) GetID() int {
	return cm.id
}

// GenerateChannelIdentity creates a new private channel identity
// ([channel.PrivateIdentity]) from scratch and assigns it a codename.
//
// The public component can be retrieved as JSON via
// [GetPublicChannelIdentityFromPrivate].
//
// Parameters:
//   - cmixID - ID of [Cmix] object in tracker. This can be retrieved using
//     [Cmix.GetID].
//
// Returns:
//   - Marshalled bytes of [channel.PrivateIdentity].
func GenerateChannelIdentity(cmixID int) ([]byte, error) {
	// Get user from singleton
	user, err := cmixTrackerSingleton.get(cmixID)
	if err != nil {
		return nil, err
	}

	rng := user.api.GetRng().GetStream()
	defer rng.Close()
	pi, err := cryptoChannel.GenerateIdentity(rng)
	if err != nil {
		return nil, err
	}
	return pi.Marshal(), nil
}

// ConstructIdentity creates a codename in a public [channel.Identity] from an
// extant identity for a given codeset version.
//
// Parameters:
//   - pubKey - The Ed25519 public key.
//   - codesetVersion - The version of the codeset used to generate the
//     identity.
//
// Returns:
//   - JSON of [channel.Identity].
func ConstructIdentity(pubKey []byte, codesetVersion int) ([]byte, error) {
	identity, err := cryptoChannel.ConstructIdentity(
		pubKey, uint8(codesetVersion))
	if err != nil {
		return nil, err
	}
	return json.Marshal(identity)
}

// ImportPrivateIdentity generates a new [channel.PrivateIdentity] from exported
// data.
//
// Parameters:
//   - password - The password used to encrypt the identity.
//   - data - The encrypted data from [ChannelsManager.ExportPrivateIdentity].
//
// Returns:
//   - JSON of [channel.PrivateIdentity].
func ImportPrivateIdentity(password string, data []byte) ([]byte, error) {
	pi, err := cryptoChannel.ImportPrivateIdentity(password, data)
	if err != nil {
		return nil, err
	}
	return pi.Marshal(), nil
}

// GetPublicChannelIdentity constructs a public identity ([channel.Identity])
// from a bytes version and returns it JSON marshaled.
//
// Parameters:
//   - marshaledPublic - Bytes of the public identity ([channel.Identity]).
//
// Returns:
//   - JSON of the constructed [channel.Identity].
func GetPublicChannelIdentity(marshaledPublic []byte) ([]byte, error) {
	i, err := cryptoChannel.UnmarshalIdentity(marshaledPublic)
	if err != nil {
		return nil, err
	}
	return json.Marshal(&i)
}

// GetPublicChannelIdentityFromPrivate returns the public identity
// ([channel.Identity]) contained in the given private identity
// ([channel.PrivateIdentity]).
//
// Parameters:
//   - marshaledPrivate - Marshaled bytes of the private identity
//     ([channel.PrivateIdentity]).
//
// Returns:
//   - JSON of the public [channel.Identity].
func GetPublicChannelIdentityFromPrivate(marshaledPrivate []byte) ([]byte, error) {
	pi, err := cryptoChannel.UnmarshalPrivateIdentity(marshaledPrivate)
	if err != nil {
		return nil, err
	}
	return json.Marshal(&pi.Identity)
}

// NewChannelsManagerMobile creates a new [ChannelsManager] from a new private
// identity [cryptoChannel.PrivateIdentity] backed with SqlLite for mobile use.
//
// This is for creating a manager for an identity for the first time. For
// generating a new one channel identity, use [GenerateChannelIdentity]. To
// reload this channel manager, use [LoadChannelsManager], passing in the
// storage tag retrieved by [ChannelsManager.GetStorageTag].
//
// Parameters:
//   - cmixID - ID of [Cmix] object in tracker. This can be retrieved using
//     [Cmix.GetID].
//   - privateIdentity - Bytes of a private identity ([channel.PrivateIdentity])
//     that is generated by [GenerateChannelIdentity].
//   - dbFilePath - absolute string path to the SqlLite database file
//   - extensionBuilderIDsJSON - JSON of an array of integers of
//     [channels.ExtensionBuilder] IDs. The ID can be retrieved from an object
//     with an extension builder (e.g.,
//     [ChannelsFileTransfer.GetExtensionBuilderID]). Leave empty if not using
//     extension builders. Example: `[2,11,5]`.
//   - cipherID - ID of [ChannelDbCipher] object in tracker.
//   - notificationsID - ID of [Notifications] object in tracker. This can be
//     retrieved using [Notifications.GetID].
//   - uiCallbacks - Callbacks to inform the UI about various events. The entire
//     interface can be nil, but if defined, each method must be implemented.
func NewChannelsManagerMobile(cmixID int, privateIdentity []byte,
	dbFilePath string, extensionBuilderIDsJSON []byte, cipherID,
	notificationsID int, uiCallbacks ChannelUICallbacks) (*ChannelsManager, error) {
	pi, err := cryptoChannel.UnmarshalPrivateIdentity(privateIdentity)
	if err != nil {
		return nil, err
	}

	// Get managers from singletons
	user, err := cmixTrackerSingleton.get(cmixID)
	if err != nil {
		return nil, err
	}
	cipher, err := channelDbCipherTrackerSingleton.get(cipherID)
	if err != nil {
		return nil, err
	}
	notif, err := notifTrackerSingleton.get(notificationsID)
	if err != nil {
		return nil, err
	}

	// Load extension builders from singleton
	extensionBuilders, err := loadExtensionBuilders(extensionBuilderIDsJSON)
	if err != nil {
		return nil, err
	}

	wrap := newChannelUICallbacksWrapper(uiCallbacks)

	model, err := storage.NewEventModel(dbFilePath, cipher,
		wrap.MessageReceived, wrap.MessageDeleted, wrap.UserMuted)
	if err != nil {
		return nil, err
	}

	channelsKV, err := user.api.GetStorage().GetKV().Prefix("channels")
	if err != nil {
		return nil, err
	}

	// Construct new channels manager
	m, err := channels.NewManager(pi, channelsKV, user.api.GetCmix(),
		user.api.GetRng(), model, extensionBuilders, user.api.AddService,
		notif.manager, wrap)
	if err != nil {
		return nil, err
	}

	// Add channel to singleton and return
	return channelManagerTrackerSingleton.make(m), nil
}

// LoadChannelsManagerMobile loads an existing [ChannelsManager] for the given
// storage tag backed with SqlLite for mobile use.
//
// This is for loading a manager for an identity that has already been created.
// The channel manager should have previously been created with
// [NewChannelsManagerMobile] and the storage is retrievable with
// [ChannelsManager.GetStorageTag].
//
// Parameters:
//   - cmixID - ID of [Cmix] object in tracker. This can be retrieved using
//     [Cmix.GetID].
//   - storageTag - The storage tag associated with the previously created
//     channel manager and retrieved with [ChannelsManager.GetStorageTag].
//   - dbFilePath - absolute string path to the SqlLite database file
//   - cipherID - ID of [ChannelDbCipher] object in tracker.
//   - extensionBuilderIDsJSON - JSON of an array of integers of
//     [channels.ExtensionBuilder] IDs. The ID can be retrieved from an object
//     with an extension builder (e.g.,
//     [ChannelsFileTransfer.GetExtensionBuilderID]). Leave empty if not using
//     extension builders. Example: `[2,11,5]`.
//   - notificationsID - ID of [Notifications] object in tracker. This can be
//     retrieved using [Notifications.GetID].
//   - uiCallbacks - Callbacks to inform the UI about various events. The entire
//     interface can be nil, but if defined, each method must be implemented.
func LoadChannelsManagerMobile(cmixID int, storageTag, dbFilePath string,
	cipherID int, extensionBuilderIDsJSON []byte, notificationsID int,
	uiCallbacks ChannelUICallbacks) (*ChannelsManager, error) {

	// Get user from singleton
	user, err := cmixTrackerSingleton.get(cmixID)
	if err != nil {
		return nil, err
	}
	cipher, err := channelDbCipherTrackerSingleton.get(cipherID)
	if err != nil {
		return nil, err
	}
	notif, err := notifTrackerSingleton.get(notificationsID)
	if err != nil {
		return nil, err
	}

	wrap := newChannelUICallbacksWrapper(uiCallbacks)

	model, err := storage.NewEventModel(dbFilePath, cipher,
		wrap.MessageReceived, wrap.MessageDeleted, wrap.UserMuted)
	if err != nil {
		return nil, err
	}

	channelsKV, err := user.api.GetStorage().GetKV().Prefix("channels")
	if err != nil {
		return nil, err
	}

	// Load extension builders from singleton
	extensionBuilders, err := loadExtensionBuilders(extensionBuilderIDsJSON)
	if err != nil {
		return nil, err
	}

	// Construct new channels manager
	m, err := channels.LoadManager(storageTag, channelsKV, user.api.GetCmix(),
		user.api.GetRng(), model, extensionBuilders, notif.manager, wrap)
	if err != nil {
		return nil, err
	}

	// Add channel to singleton and return
	return channelManagerTrackerSingleton.make(m), nil
}

// NewChannelsManager creates a new [ChannelsManager] from a new private
// identity [channel.PrivateIdentity].
//
// This is for creating a manager for an identity for the first time. For
// generating a new one channel identity, use [GenerateChannelIdentity]. To
// reload this channel manager, use [LoadChannelsManager], passing in the
// storage tag retrieved by [ChannelsManager.GetStorageTag].
//
// Parameters:
//   - cmixID - ID of [Cmix] object in tracker. This can be retrieved using
//     [Cmix.GetID].
//   - privateIdentity - Bytes of a private identity ([channel.PrivateIdentity])
//     that is generated by [GenerateChannelIdentity].
//   - eventBuilder - An interface that contains a function that initialises and
//     returns the event model that is bindings-compatible.
//   - extensionBuilderIDsJSON - JSON of an array of integers of
//     [channels.ExtensionBuilder] IDs. The ID can be retrieved from an object
//     with an extension builder (e.g.,
//     [ChannelsFileTransfer.GetExtensionBuilderID]). Leave empty if not using
//     extension builders. Example: `[2,11,5]`.
//   - notificationsID - ID of [Notifications] object in tracker. This can be
//     retrieved using [Notifications.GetID].
//   - uiCallbacks - Callbacks to inform the UI about various events. The entire
//     interface can be nil, but if defined, each method must be implemented.
func NewChannelsManager(cmixID int, privateIdentity []byte,
	eventBuilder EventModelBuilder, extensionBuilderIDsJSON []byte,
	notificationsID int, uiCallbacks ChannelUICallbacks) (*ChannelsManager, error) {
	pi, err := cryptoChannel.UnmarshalPrivateIdentity(privateIdentity)
	if err != nil {
		return nil, err
	}

	// Get managers from singletons
	user, err := cmixTrackerSingleton.get(cmixID)
	if err != nil {
		return nil, err
	}
	notif, err := notifTrackerSingleton.get(notificationsID)
	if err != nil {
		return nil, err
	}

	// Load extension builders from singleton
	extensionBuilders, err := loadExtensionBuilders(extensionBuilderIDsJSON)
	if err != nil {
		return nil, err
	}

	eb := func(path string) (channels.EventModel, error) {
		return NewEventModel(eventBuilder.Build(path)), nil
	}

	wrap := newChannelUICallbacksWrapper(uiCallbacks)

	channelsKV, err := user.api.GetStorage().GetKV().Prefix("channels")
	if err != nil {
		return nil, err
	}

	// Construct new channels manager
	m, err := channels.NewManagerBuilder(pi, channelsKV, user.api.GetCmix(),
		user.api.GetRng(), eb, extensionBuilders, user.api.AddService,
		notif.manager, wrap)
	if err != nil {
		return nil, err
	}

	// Add channel to singleton and return
	return channelManagerTrackerSingleton.make(m), nil
}

// LoadChannelsManager loads an existing [ChannelsManager] for the given storage
// tag.
//
// This is for loading a manager for an identity that has already been created.
// The channel manager should have previously been created with
// [NewChannelsManager] and the storage is retrievable with
// [ChannelsManager.GetStorageTag].
//
// Parameters:
//   - cmixID - ID of [Cmix] object in tracker. This can be retrieved using
//     [Cmix.GetID].
//   - storageTag - The storage tag associated with the previously created
//     channel manager and retrieved with [ChannelsManager.GetStorageTag].
//   - event - An interface that contains a function that initializes and
//     returns the event model that is bindings-compatible.
//   - extensionBuilderIDsJSON - JSON of an array of integers of
//     [channels.ExtensionBuilder] IDs. The ID can be retrieved from an object
//     with an extension builder (e.g.,
//     [ChannelsFileTransfer.GetExtensionBuilderID]). Leave empty if not using
//     extension builders. Example: `[2,11,5]`.
//   - notificationsID - ID of [Notifications] object in tracker. This can be
//     retrieved using [Notifications.GetID].
//   - uiCallbacks - Callbacks to inform the UI about various events. The entire
//     interface can be nil, but if defined, each method must be implemented.
func LoadChannelsManager(cmixID int, storageTag string,
	eventBuilder EventModelBuilder, extensionBuilderIDsJSON []byte,
	notificationsID int, uiCallbacks ChannelUICallbacks) (*ChannelsManager, error) {

	// Get managers from singletons
	user, err := cmixTrackerSingleton.get(cmixID)
	if err != nil {
		return nil, err
	}
	notif, err := notifTrackerSingleton.get(notificationsID)
	if err != nil {
		return nil, err
	}

	eb := func(path string) (channels.EventModel, error) {
		return NewEventModel(eventBuilder.Build(path)), nil
	}

	// Load extension builders from singleton
	extensionBuilders, err := loadExtensionBuilders(extensionBuilderIDsJSON)
	if err != nil {
		return nil, err
	}

	wrap := newChannelUICallbacksWrapper(uiCallbacks)

	channelsKV, err := user.api.GetStorage().GetKV().Prefix("channels")
	if err != nil {
		return nil, err
	}

	// Construct new channels manager
	m, err := channels.LoadManagerBuilder(storageTag, channelsKV,
		user.api.GetCmix(), user.api.GetRng(), eb, extensionBuilders,
		notif.manager, wrap)
	if err != nil {
		return nil, err
	}

	// Add channel to singleton and return
	return channelManagerTrackerSingleton.make(m), nil
}

// NewChannelsManagerGoEventModel creates a new [ChannelsManager] from a new
// private identity ([channel.PrivateIdentity]). This is not compatible with
// GoMobile Bindings because it receives the go event model.
//
// This is for creating a manager for an identity for the first time. For
// generating a new one channel identity, use [GenerateChannelIdentity]. To
// reload this channel manager, use [LoadChannelsManagerGoEventModel], passing
// in the storage tag retrieved by [ChannelsManager.GetStorageTag].
//
// Parameters:
//   - cmixID - ID of [Cmix] object in tracker. This can be retrieved using
//     [Cmix.GetID].
//   - privateIdentity - Bytes of a private identity ([channel.PrivateIdentity])
//     that is generated by [GenerateChannelIdentity].
//   - goEventBuilder - A function that initializes and returns the event model
//     that is not compatible with GoMobile bindings.
//   - extensionBuilderIDsJSON - JSON of an array of integers of
//     [channels.ExtensionBuilder] IDs. The ID can be retrieved from an object
//     with an extension builder (e.g.,
//     [ChannelsFileTransfer.GetExtensionBuilderID]). Leave empty if not using
//     extension builders. Example: `[2,11,5]`.
//   - notificationsID - ID of [Notifications] object in tracker. This can be
//     retrieved using [Notifications.GetID].
//   - uiCallbacks - Callbacks to inform the UI about various events. The entire
//     interface can be nil, but if defined, each method must be implemented.
func NewChannelsManagerGoEventModel(cmixID int, privateIdentity,
	extensionBuilderIDsJSON []byte, goEventBuilder channels.EventModelBuilder,
	notificationsID int, callbacks ChannelUICallbacks) (
	*ChannelsManager, error) {
	pi, err := cryptoChannel.UnmarshalPrivateIdentity(privateIdentity)
	if err != nil {
		return nil, err
	}

	// Get managers from singletons
	user, err := cmixTrackerSingleton.get(cmixID)
	if err != nil {
		return nil, err
	}
	notif, err := notifTrackerSingleton.get(notificationsID)
	if err != nil {
		return nil, err
	}

	// Load extension builders from singleton
	extensionBuilders, err := loadExtensionBuilders(extensionBuilderIDsJSON)
	if err != nil {
		return nil, err
	}

	cbs := newChannelUICallbacksWrapper(callbacks)
	channelsKV, err := user.api.GetStorage().GetKV().Prefix("channels")
	if err != nil {
		return nil, err
	}

	// Construct new channels manager
	m, err := channels.NewManagerBuilder(pi, channelsKV, user.api.GetCmix(),
		user.api.GetRng(), goEventBuilder, extensionBuilders,
		user.api.AddService, notif.manager, cbs)
	if err != nil {
		return nil, err
	}

	// Add channel to singleton and return
	return channelManagerTrackerSingleton.make(m), nil
}

// LoadChannelsManagerGoEventModel loads an existing ChannelsManager. This is
// not compatible with GoMobile Bindings because it receives the go event model.
// This is for creating a manager for an identity for the first time. The
// channel manager should have first been created with
// NewChannelsManagerGoEventModel and then the storage tag can be retrieved
// with ChannelsManager.GetStorageTag
//
// Parameters:
//   - cmixID - ID of [Cmix] object in tracker. This can be retrieved using
//     [Cmix.GetID].
//   - storageTag - retrieved with ChannelsManager.GetStorageTag
//   - goEvent - A function that initializes and returns the event model that is
//     not compatible with GoMobile bindings.
//   - extensionBuilderIDsJSON - JSON of an array of integers of
//     [channels.ExtensionBuilder] IDs. The ID can be retrieved from an object
//     with an extension builder (e.g.,
//     [ChannelsFileTransfer.GetExtensionBuilderID]). Leave empty if not using
//     extension builders. Example: `[2,11,5]`.
//   - notificationsID - ID of [Notifications] object in tracker. This can be
//     retrieved using [Notifications.GetID].
//   - uiCallbacks - Callbacks to inform the UI about various events. The entire
//     interface can be nil, but if defined, each method must be implemented.
func LoadChannelsManagerGoEventModel(cmixID int, storageTag string,
	goEventBuilder channels.EventModelBuilder, extensionBuilderIDsJSON []byte,
	notificationsID int, uiCallbacks ChannelUICallbacks) (*ChannelsManager, error) {

	// Get managers from singletons
	user, err := cmixTrackerSingleton.get(cmixID)
	if err != nil {
		return nil, err
	}
	notif, err := notifTrackerSingleton.get(notificationsID)
	if err != nil {
		return nil, err
	}

	cbs := newChannelUICallbacksWrapper(uiCallbacks)
	channelsKV, err := user.api.GetStorage().GetKV().Prefix("channels")
	if err != nil {
		return nil, err
	}

	// Load extension builders from singleton
	extensionBuilders, err := loadExtensionBuilders(extensionBuilderIDsJSON)
	if err != nil {
		return nil, err
	}

	// Construct new channels manager
	m, err := channels.LoadManagerBuilder(storageTag, channelsKV,
		user.api.GetCmix(), user.api.GetRng(), goEventBuilder,
		extensionBuilders, notif.manager, cbs)
	if err != nil {
		return nil, err
	}

	// Add channel to singleton and return
	return channelManagerTrackerSingleton.make(m), nil
}

// loadExtensionBuilders unmarshalls the list of extension builder IDs and
// loads them from the singleton.
func loadExtensionBuilders(
	extensionBuilderIDsJSON []byte) ([]channels.ExtensionBuilder, error) {
	if len(extensionBuilderIDsJSON) != 0 {
		var ebIDS []int
		if err := json.Unmarshal(extensionBuilderIDsJSON, &ebIDS); err != nil {
			return nil, err
		}
		return channelExtensionBuilderTrackerSingleton.get(ebIDS...), nil
	}
	return []channels.ExtensionBuilder{}, nil
}

////////////////////////////////////////////////////////////////////////////////
// Channel Actions                                                            //
////////////////////////////////////////////////////////////////////////////////

// DecodePublicURL decodes the channel URL into a channel pretty print. This
// function can only be used for public channel URLs. To get the privacy level
// of a channel URL, use [GetShareUrlType].
//
// Parameters:
//   - url - The channel's share URL. Should be received from another user or
//     generated via [GetShareURL].
//
// Returns:
//   - The channel pretty print.
func DecodePublicURL(url string) (string, error) {
	c, err := cryptoBroadcast.DecodeShareURL(url, "")
	if err != nil {
		return "", err
	}

	return c.PrettyPrint(), nil
}

// DecodePrivateURL decodes the channel URL, using the password, into a channel
// pretty print. This function can only be used for private or secret channel
// URLs. To get the privacy level of a channel URL, use [GetShareUrlType].
//
// Parameters:
//   - url - The channel's share URL. Should be received from another user or
//     generated via [GetShareURL].
//   - password - The password needed to decrypt the secret data in the URL.
//
// Returns:
//   - The channel pretty print.
func DecodePrivateURL(url, password string) (string, error) {
	c, err := cryptoBroadcast.DecodeShareURL(url, password)
	if err != nil {
		return "", err
	}

	return c.PrettyPrint(), nil
}

// GetChannelJSON returns the JSON of the channel for the given pretty print.
//
// Parameters:
//   - prettyPrint - The pretty print of the channel.
//
// Returns:
//   - JSON of the [broadcast.Channel] object.
//
// Example JSON of [broadcast.Channel]:
//
//	{
//	  "ReceptionID": "Ja/+Jh+1IXZYUOn+IzE3Fw/VqHOscomD0Q35p4Ai//kD",
//	  "Name": "My_Channel",
//	  "Description": "Here is information about my channel.",
//	  "Salt": "+tlrU/htO6rrV3UFDfpQALUiuelFZ+Cw9eZCwqRHk+g=",
//	  "RsaPubKeyHash": "PViT1mYkGBj6AYmE803O2RpA7BX24EjgBdldu3pIm4o=",
//	  "RsaPubKeyLength": 5,
//	  "RSASubPayloads": 1,
//	  "Secret": "JxZt/wPx2luoPdHY6jwbXqNlKnixVU/oa9DgypZOuyI=",
//	  "Level": 0
//	}
func GetChannelJSON(prettyPrint string) ([]byte, error) {
	c, err := cryptoBroadcast.NewChannelFromPrettyPrint(prettyPrint)
	if err != nil {
		return nil, nil
	}

	return json.Marshal(c)
}

// ChannelInfo contains information about a channel.
//
// Example of ChannelInfo JSON:
//
//	{
//	  "Name": "Test Channel",
//	  "Description": "This is a test channel",
//	  "ChannelID": "RRnpRhmvXtW9ugS1nILJ3WfttdctDvC2jeuH43E0g/0D",
//	}
type ChannelInfo struct {
	Name        string
	Description string
	ChannelID   string
}

// GetChannelInfo returns the info about a channel from its public description.
//
// Parameters:
//   - prettyPrint - The pretty print of the channel.
//
// The pretty print will be of the format:
//
//	<Speakeasy-v3:Test_Channel|description:Channel description.|level:Public|created:1666718081766741100|secrets:+oHcqDbJPZaT3xD5NcdLY8OjOMtSQNKdKgLPmr7ugdU=|rCI0wr01dHFStjSFMvsBzFZClvDIrHLL5xbCOPaUOJ0=|493|1|7cBhJxVfQxWo+DypOISRpeWdQBhuQpAZtUbQHjBm8NQ=>
//
// Returns:
//   - []byte - JSON of [ChannelInfo], which describes all relevant channel
//     info.
func GetChannelInfo(prettyPrint string) ([]byte, error) {
	_, bytes, err := getChannelInfo(prettyPrint)
	return bytes, err
}

func getChannelInfo(prettyPrint string) (*cryptoBroadcast.Channel, []byte, error) {
	c, err := cryptoBroadcast.NewChannelFromPrettyPrint(prettyPrint)
	if err != nil {
		return nil, nil, err
	}
	ci := &ChannelInfo{
		Name:        c.Name,
		Description: c.Description,
		ChannelID:   c.ReceptionID.String(),
	}
	bytes, err := json.Marshal(ci)
	if err != nil {
		return nil, nil, err
	}
	return c, bytes, nil
}

// GenerateChannel creates a new channel with the user as the admin and returns
// the broadcast.Channel object. This function only create a channel and does
// not join it.
//
// The private key is saved to storage and can be accessed with
// ExportChannelAdminKey.
//
// Parameters:
//   - name - The name of the new channel. The name must be between 3 and 24
//     characters inclusive. It can only include upper and lowercase Unicode
//     letters, digits 0 through 9, and underscores (_). It cannot be
//     changed once a channel is created.
//   - description - The description of a channel. The description is optional
//     but cannot be longer than 144 characters and can include all Unicode
//     characters. It cannot be changed once a channel is created.
//   - privacyLevel - The [broadcast.PrivacyLevel] of the channel. 0 = public,
//     1 = private, and 2 = secret. Refer to the comment below for more
//     information.
//
// Returns:
//   - string - The pretty print of the channel.
//
// The [broadcast.PrivacyLevel] of a channel indicates the level of channel
// information revealed when sharing it via URL. For any channel besides public
// channels, the secret information is encrypted and a password is required to
// share and join a channel.
//   - A privacy level of [broadcast.Public] reveals all the information
//     including the name, description, privacy level, public key and salt.
//   - A privacy level of [broadcast.Private] reveals only the name and
//     description.
//   - A privacy level of [broadcast.Secret] reveals nothing.
func (cm *ChannelsManager) GenerateChannel(
	name, description string, privacyLevel int) (string, error) {
	level := cryptoBroadcast.PrivacyLevel(privacyLevel)
	ch, err := cm.api.GenerateChannel(name, description, level)
	if err != nil {
		return "", err
	}

	return ch.PrettyPrint(), nil
}

// JoinChannel joins the given channel. It will return the error
// [channels.ChannelAlreadyExistsErr] if the channel has already been joined.
//
// Parameters:
//   - channelPretty - A portable channel string. Should be received from
//     another user or generated via [ChannelsManager.GenerateChannel].
//
// The pretty print will be of the format:
//
//	<Speakeasy-v3:Test_Channel|description:Channel description.|level:Public|created:1666718081766741100|secrets:+oHcqDbJPZaT3xD5NcdLY8OjOMtSQNKdKgLPmr7ugdU=|rCI0wr01dHFStjSFMvsBzFZClvDIrHLL5xbCOPaUOJ0=|493|1|7cBhJxVfQxWo+DypOISRpeWdQBhuQpAZtUbQHjBm8NQ=>
//
// Returns:
//   - []byte - JSON of [ChannelInfo], which describes all relevant channel
//     info.
func (cm *ChannelsManager) JoinChannel(channelPretty string) ([]byte, error) {
	c, info, err := getChannelInfo(channelPretty)
	if err != nil {
		return nil, err
	}

	// Join the channel using the API
	err = cm.api.JoinChannel(c)

	return info, err
}

// LeaveChannel leaves the given channel. It will return the error
// [channels.ChannelDoesNotExistsErr] if the channel was not previously joined.
//
// Parameters:
//   - channelIdBytes - Marshalled bytes of the channel's [id.ID].
func (cm *ChannelsManager) LeaveChannel(channelIdBytes []byte) error {
	// Unmarshal channel ID
	channelID, err := id.Unmarshal(channelIdBytes)
	if err != nil {
		return err
	}

	// Leave the channel
	return cm.api.LeaveChannel(channelID)
}

// ReplayChannel replays all messages from the channel within the network's
// memory (~3 weeks) over the event model.
//
// Returns the error [channels.ChannelDoesNotExistsErr] if the channel was not
// previously joined.
//
// Parameters:
//   - channelIdBytes - Marshalled bytes of the channel's [id.ID].
func (cm *ChannelsManager) ReplayChannel(channelIdBytes []byte) error {
	// Unmarshal channel ID
	channelID, err := id.Unmarshal(channelIdBytes)
	if err != nil {
		return err
	}

	// Replay channel
	return cm.api.ReplayChannel(channelID)
}

// GetChannels returns the IDs of all channels that have been joined.
//
// Returns:
//   - []byte - A JSON marshalled array of [id.ID].
//
// JSON Example:
//
//	{
//	  "U4x/lrFkvxuXu59LtHLon1sUhPJSCcnZND6SugndnVID",
//	  "15tNdkKbYXoMn58NO6VbDMDWFEyIhTWEGsvgcJsHWAgD"
//	}
func (cm *ChannelsManager) GetChannels() ([]byte, error) {
	channelIds := cm.api.GetChannels()
	return json.Marshal(channelIds)
}

// EnableDirectMessages enables the token for direct messaging for this
// channel.
//
// Parameters:
//   - channelIdBytes - Marshalled bytes of the channel's [id.ID].
func (cm *ChannelsManager) EnableDirectMessages(channelIdBytes []byte) error {
	channelID, err := id.Unmarshal(channelIdBytes)
	if err != nil {
		return err
	}

	return cm.api.EnableDirectMessages(channelID)
}

// DisableDirectMessages removes the token for direct messaging for a
// given channel.
//
// Parameters:
//   - channelIdBytes - Marshalled bytes of the channel's [id.ID].
func (cm *ChannelsManager) DisableDirectMessages(channelIdBytes []byte) error {
	channelID, err := id.Unmarshal(channelIdBytes)
	if err != nil {
		return err
	}

	return cm.api.DisableDirectMessages(channelID)
}

// AreDMsEnabled returns the status of DMs for a given channel
//
// Parameters:
//   - channelIdBytes - Marshalled bytes of the channel's [id.ID].
//
// Returns:
//   - DM status (bool) - true if DMs are enabled for passed in channel
func (cm *ChannelsManager) AreDMsEnabled(channelIdBytes []byte) (bool, error) {
	channelID, err := id.Unmarshal(channelIdBytes)
	if err != nil {
		return false, err
	}
	return cm.api.AreDMsEnabled(channelID), nil
}

////////////////////////////////////////////////////////////////////////////////
// Channel Share URL                                                          //
////////////////////////////////////////////////////////////////////////////////

// ShareURL is returned from ChannelsManager.GetShareURL. It includes the
// channel's share URL and password, if it needs one.
//
// JSON example for a public channel:
//
//	{
//	  "url": "https://internet.speakeasy.tech/?0Name=name&1Description=desc&2Level=Public&3Created=1665489600000000000&e=%2FWNZvuHPuv%2Bx23XbZXVNzCi7y8rUSxkh75MpR9UrsCo%3D&k=ddX1CH52xH%2F%2Fb6lKrbvDghdSmCQr90ktsOAZ%2FrhEonI%3D&l=2&m=0&p=328&s=%2FD%2FoQP2mio3XAWfhmWF0xmZrpj4nAsb9JLXj%2B0Mzq9Y%3D&v=1",
//	  "password": ""
//	}
//
// JSON example for a private channel:
//
//	{
//	  "url": "https://internet.speakeasy.tech/?0Name=name&1Description=desc&3Created=1665489600000000000&d=5AZQirb%2FYrmUITLn%2FFzCaGek1APfJnd2q0KwORGj%2BnbGg26kTShG6cfD3w6c%2BA3RDzxuKDSDN0zS4n1LbjiGe0KYdb8eJVeyRZtld516hfojNDXNAwZq8zbeZy4jjbF627fcLHRNS%2FaII4uJ5UB3gLUeBeZGraaybCCu3FIj1N4RbcJ5cQgT7hBf93bHmJc%3D&m=0&v=1",
//	  "password": "tribune gangrene labrador italics nutmeg process exhume legal"
//	}
//
// JSON example for a secret channel:
//
//	{
//	  "url": "https://internet.speakeasy.tech/?d=w5evLthm%2Fq2j11g6PPtV0QoLaAqNCIER0OqxhxL%2FhpGVJI0057ZPgGBrKoJNE1%2FdoVuU35%2FhohuW%2BWvGlx6IuHoN6mDj0HfNj6Lo%2B8GwIaD6jOEwUcH%2FMKGsKnoqFsMaMPd5gXYgdHvA8l5SRe0gSCVqGKUaG6JgL%2FDu4iyjY7v4ykwZdQ7soWOcBLHDixGEkVLpwsCrPVHkT2K0W6gV74GIrQ%3D%3D&m=0&v=1",
//	  "password": "frenzy contort staple thicket consuming affiliate scion demeanor"
//	}
type ShareURL struct {
	URL      string `json:"url"`
	Password string `json:"password"`
}

// GetShareURL generates a URL that can be used to share this channel with
// others on the given host.
//
// A URL comes in one of three forms based on the privacy level set when
// generating the channel. Each privacy level hides more information than the
// last with the lowest level revealing everything and the highest level
// revealing nothing. For any level above the lowest, a password is returned,
// which will be required when decoding the URL.
//
// The maxUses is the maximum number of times this URL can be used to join a
// channel. If it is set to 0, then it can be shared unlimited times. The max
// uses is set as a URL parameter using the key [broadcast.MaxUsesKey]. Note
// that this number is also encoded in the secret data for private and secret
// URLs, so if the number is changed in the URL, it will be verified when
// calling [DecodePublicURL] and [DecodePrivateURL]. There is no enforcement for
// public URLs.
//
// Parameters:
//   - cmixID - ID of [Cmix] object in tracker.
//   - host - The URL to append the channel info to.
//   - maxUses - The maximum number of uses the link can be used (0 for
//     unlimited).
//   - channelIdBytes - Marshalled bytes of the channel ([id.ID]).
//
// Returns:
//   - JSON of [ShareURL].
func (cm *ChannelsManager) GetShareURL(cmixID int, host string, maxUses int,
	channelIdBytes []byte) ([]byte, error) {

	// Unmarshal channel ID
	channelID, err := id.Unmarshal(channelIdBytes)
	if err != nil {
		return nil, err
	}

	// Get the channel from the ID
	ch, err := cm.api.GetChannel(channelID)
	if err != nil {
		return nil, err
	}

	// Get user from singleton
	user, err := cmixTrackerSingleton.get(cmixID)
	if err != nil {
		return nil, err
	}

	// Generate share URL and password
	rng := user.api.GetRng().GetStream()
	url, password, err := ch.ShareURL(host, maxUses, rng)
	rng.Close()
	if err != nil {
		return nil, err
	}

	su := ShareURL{
		URL:      url,
		Password: password,
	}

	return json.Marshal(su)
}

// GetShareUrlType determines the [broadcast.PrivacyLevel] of the channel URL.
// If the URL is an invalid channel URL, an error is returned.
//
// Parameters:
//   - url - The channel share URL.
//
// Returns:
//   - An int that corresponds to the [broadcast.PrivacyLevel] as outlined
//     below.
//
// Possible returns:
//
//	0 = public channel
//	1 = private channel
//	2 = secret channel
func GetShareUrlType(url string) (int, error) {
	level, err := cryptoBroadcast.GetShareUrlType(url)
	return int(level), err
}

////////////////////////////////////////////////////////////////////////////////
// Channel Sending Methods & Reports                                          //
////////////////////////////////////////////////////////////////////////////////

// ChannelSendReport is the bindings' representation of the return values of
// [ChannelsManager]'s send operations. The messageID and ephID may not be
// present.
//
// JSON Example:
//
//	{
//	  "messageID": "0kitNxoFdsF4q1VMSI/xPzfCnGB2l+ln2+7CTHjHbJw=",
//	  "roundsList":[1,5,9],
//	  "ephId": 0
//	}
type ChannelSendReport struct {
	MessageID  []byte `json:"messageID,omitempty"`
	RoundsList `json:"roundsList"`
	EphId      int64 `json:"ephId,omitempty"`
}

// ValidForeverBindings is the value used to represent the maximum time a
// message can be valid for when used over the bindings.
const ValidForeverBindings = -1

// ValidForever returns the value to use for validUntil when you want a message
// to be available for the maximum amount of time.
func ValidForever() int {
	// ValidForeverBindings is returned instead of channels.ValidForever,
	// because the latter can cause an integer overflow
	return ValidForeverBindings
}

// SendGeneric is used to send a raw message over a channel. In general, it
// should be wrapped in a function that defines the wire protocol.
//
// If the final message, before being sent over the wire, is too long, this
// will return an error. Due to the underlying encoding using compression,
// it is not possible to define the largest payload that can be sent, but it
// will always be possible to send a payload of 802 bytes at minimum.
//
// Parameters:
//   - channelIdBytes - Marshalled bytes of the channel's [id.ID].
//   - messageType - The message type of the message. This will be a valid
//     [channels.MessageType].
//   - message - The contents of the message. This need not be of data type
//     string, as the message could be a specified format that the channel may
//     recognize.
//   - validUntilMS - The lease of the message. This will be how long the
//     message is available from the network, in milliseconds. As per the
//     [channels.Manager] documentation, this has different meanings depending
//     on the use case. These use cases may be generic enough that they will not
//     be enumerated here. Use [channels.ValidForever] to last the max message
//     life.
//   - tracked - Set tracked to true if the message should be tracked in the
//     sendTracker, which allows messages to be shown locally before they are
//     received on the network. In general, all messages that will be displayed
//     to the user should be tracked while all actions should not be.
//   - cmixParamsJSON - A JSON marshalled [xxdk.CMIXParams]. This may be empty,
//     and [GetDefaultCMixParams] will be used internally.
//   - pingsJSON - JSON of a slice of public keys of users that should receive
//     mobile notifications for the message.
//
// Example pingsJSON:
//
//	[
//	  "FgJMvgSsY4rrKkS/jSe+vFOJOs5qSSyOUSW7UtF9/KU=",
//	  "fPqcHtrJ398PAC35QyWXEU9PHzz8Z4BKQTCxSvpSygw=",
//	  "JnjCgh7g/+hNiI9VPKW01aRSxGOFmNulNCymy3ImXAo="
//	]
//
// Returns:
//   - []byte - JSON of [ChannelSendReport].
func (cm *ChannelsManager) SendGeneric(channelIdBytes []byte, messageType int,
	message []byte, validUntilMS int64, tracked bool, cmixParamsJSON []byte,
	pingsJSON []byte) ([]byte, error) {

	// Unmarshal channel ID and parameters
	channelID, params, err :=
		parseChannelsParameters(channelIdBytes, cmixParamsJSON)
	if err != nil {
		return nil, err
	}

	msgType := channels.MessageType(messageType)

	// Calculate lease
	lease := time.Duration(validUntilMS) * time.Millisecond
	if validUntilMS == ValidForeverBindings {
		lease = channels.ValidForever
	}

	var pings []ed25519.PublicKey
	if err = json.Unmarshal(pingsJSON, &pings); err != nil {
		return nil, err
	}

	// Send message
	messageID, rnd, ephID, err := cm.api.SendGeneric(
		channelID, msgType, message, lease, tracked, params.CMIX, pings)
	if err != nil {
		return nil, err
	}

	// Construct send report
	return constructChannelSendReport(&messageID, rnd.ID, &ephID)
}

// SendMessage is used to send a formatted message over a channel.
//
// Due to the underlying encoding using compression, it isn't possible to define
// the largest payload that can be sent, but it will always be possible to send
// a payload of 798 bytes at minimum.
//
// The message will auto delete validUntil after the round it is sent in,
// lasting forever if [channels.ValidForever] is used.
//
// Parameters:
//   - channelIdBytes - Marshalled bytes of the channel's [id.ID].
//   - message - The contents of the message. The message should be at most 510
//     bytes. This is expected to be Unicode, and thus a string data type is
//     expected
//   - validUntilMS - The lease of the message. This will be how long the
//     message is available from the network, in milliseconds. As per the
//     [channels.Manager] documentation, this has different meanings depending
//     on the use case. These use cases may be generic enough that they will not
//     be enumerated here. Use [channels.ValidForever] to last the max message
//     life.
//   - cmixParamsJSON - A JSON marshalled [xxdk.CMIXParams]. This may be
//     empty, and [GetDefaultCMixParams] will be used internally.
//   - pingsJSON - JSON of a slice of public keys of users that should receive
//     mobile notifications for the message.
//
// Example pingsJSON:
//
//	[
//	  "FgJMvgSsY4rrKkS/jSe+vFOJOs5qSSyOUSW7UtF9/KU=",
//	  "fPqcHtrJ398PAC35QyWXEU9PHzz8Z4BKQTCxSvpSygw=",
//	  "JnjCgh7g/+hNiI9VPKW01aRSxGOFmNulNCymy3ImXAo="
//	]
//
// Returns:
//   - []byte - JSON of [ChannelSendReport].
func (cm *ChannelsManager) SendMessage(channelIdBytes []byte, message string,
	validUntilMS int64, cmixParamsJSON []byte, pingsJSON []byte) ([]byte, error) {

	// Unmarshal channel ID and parameters
	channelID, params, err :=
		parseChannelsParameters(channelIdBytes, cmixParamsJSON)
	if err != nil {
		return nil, err
	}

	// Calculate lease
	lease := time.Duration(validUntilMS) * time.Millisecond
	if validUntilMS == ValidForeverBindings {
		lease = channels.ValidForever
	}

	var pings []ed25519.PublicKey
	if err = json.Unmarshal(pingsJSON, &pings); err != nil {
		return nil, err
	}

	// Send message
	messageID, rnd, ephID, err :=
		cm.api.SendMessage(channelID, message, lease, params.CMIX, pings)
	if err != nil {
		return nil, err
	}

	// Construct send report
	return constructChannelSendReport(&messageID, rnd.ID, &ephID)
}

// SendReply is used to send a formatted message over a channel.
//
// Due to the underlying encoding using compression, it is not possible to
// define the largest payload that can be sent, but it will always be possible
// to send a payload of 766 bytes at minimum.
//
// If the message ID that the reply is sent to does not exist, then the other
// side will post the message as a normal message and not as a reply.
//
// Parameters:
//   - channelIdBytes - Marshalled bytes of the channel's [id.ID].
//   - message - The contents of the message. The message should be at most 510
//     bytes. This is expected to be Unicode, and thus a string data type is
//     expected.
//   - messageToReactTo - The marshalled [channel.MessageID] of the message you
//     wish to reply to. This may be found in the [ChannelSendReport] if
//     replying to your own. Alternatively, if reacting to another user's
//     message, you may retrieve it via the [ChannelMessageReceptionCallback]
//     registered using [RegisterReceiveHandler].
//   - validUntilMS - The lease of the message. This will be how long the
//     message is available from the network, in milliseconds. As per the
//     [channels.Manager] documentation, this has different meanings depending
//     on the use case. These use cases may be generic enough that they will not
//     be enumerated here. Use [channels.ValidForever] to last the max message
//     life.
//   - cmixParamsJSON - A JSON marshalled [xxdk.CMIXParams]. This may be empty,
//     and [GetDefaultCMixParams] will be used internally.
//   - pingsJSON - JSON of a slice of public keys of users that should receive
//     mobile notifications for the message.
//
// Example pingsJSON:
//
//	[
//	  "FgJMvgSsY4rrKkS/jSe+vFOJOs5qSSyOUSW7UtF9/KU=",
//	  "fPqcHtrJ398PAC35QyWXEU9PHzz8Z4BKQTCxSvpSygw=",
//	  "JnjCgh7g/+hNiI9VPKW01aRSxGOFmNulNCymy3ImXAo="
//	]
//
// Returns:
//   - []byte - JSON of [ChannelSendReport].
func (cm *ChannelsManager) SendReply(channelIdBytes []byte, message string,
	messageToReactTo []byte, validUntilMS int64, cmixParamsJSON []byte,
	pingsJSON []byte) ([]byte, error) {

	// Unmarshal channel ID and parameters
	channelID, params, err :=
		parseChannelsParameters(channelIdBytes, cmixParamsJSON)
	if err != nil {
		return nil, err
	}

	// Unmarshal message ID
	messageID := cryptoMessage.ID{}
	copy(messageID[:], messageToReactTo)

	// Calculate lease
	lease := time.Duration(validUntilMS) * time.Millisecond
	if validUntilMS == ValidForeverBindings {
		lease = channels.ValidForever
	}

	var pings []ed25519.PublicKey
	if err = json.Unmarshal(pingsJSON, &pings); err != nil {
		return nil, err
	}

	// Send Reply
	messageID, rnd, ephID, err :=
		cm.api.SendReply(channelID, message, messageID, lease, params.CMIX, pings)
	if err != nil {
		return nil, err
	}

	// Construct send report
	return constructChannelSendReport(&messageID, rnd.ID, &ephID)
}

// SendReaction is used to send a reaction to a message over a channel. The
// reaction must be a single emoji with no other characters, and will be
// rejected otherwise.
//
// Clients will drop the reaction if they do not recognize the reactTo message.
//
// Parameters:
//   - channelIdBytes - Marshalled bytes of the channel's [id.ID].
//   - reaction - The user's reaction. This should be a single emoji with no
//     other characters. As such, a Unicode string is expected.
//   - messageToReactTo - The marshalled [channel.MessageID] of the message you
//     wish to reply to. This may be found in the ChannelSendReport if replying
//     to your own. Alternatively, if reacting to another user's message, you
//     may retrieve it via the ChannelMessageReceptionCallback registered using
//     RegisterReceiveHandler.
//   - validUntilMS - The lease of the message. This will be how long the
//     message is available from the network, in milliseconds. As per the
//     [channels.Manager] documentation, this has different meanings depending
//     on the use case. These use cases may be generic enough that they will not
//     be enumerated here. Use [channels.ValidForever] to last the max message
//     life.
//   - cmixParamsJSON - A JSON marshalled [xxdk.CMIXParams]. This may be empty,
//     and GetDefaultCMixParams will be used internally.
//
// Returns:
//   - []byte - JSON of [ChannelSendReport].
func (cm *ChannelsManager) SendReaction(channelIdBytes []byte, reaction string,
	messageToReactTo []byte, validUntilMS int64, cmixParamsJSON []byte) (
	[]byte, error) {

	// Unmarshal channel ID and parameters
	channelID, params, err := parseChannelsParameters(
		channelIdBytes, cmixParamsJSON)
	if err != nil {
		return nil, err
	}

	// Unmarshal message ID
	messageID := cryptoMessage.ID{}
	copy(messageID[:], messageToReactTo)

	// Calculate lease
	lease := time.Duration(validUntilMS) * time.Millisecond
	if validUntilMS == ValidForeverBindings {
		lease = channels.ValidForever
	}

	// Send reaction
	messageID, rnd, ephID, err :=
		cm.api.SendReaction(channelID, reaction, messageID, lease, params.CMIX)
	if err != nil {
		return nil, err
	}

	// Construct send report
	return constructChannelSendReport(&messageID, rnd.ID, &ephID)
}

// SendSilent is used to send to a channel a message with no notifications.
// Its primary purpose is to communicate new nicknames without calling
// [SendMessage].
//
// It takes no payload intentionally as the message should be very lightweight.
//
// Parameters:
//   - channelIdBytes - Marshalled bytes of the channel's [id.ID].
//   - validUntilMS - The lease of the message. This will be how long the
//     message is available from the network, in milliseconds. As per the
//     [channels.Manager] documentation, this has different meanings depending
//     on the use case. These use cases may be generic enough that they will not
//     be enumerated here. Use [channels.ValidForever] to last the max message
//     life.
//   - cmixParamsJSON - A JSON marshalled [xxdk.CMIXParams]. This may be empty,
//     and [GetDefaultCMixParams] will be used internally.
//
// Returns:
//   - []byte - JSON of [ChannelSendReport].
func (cm *ChannelsManager) SendSilent(channelIdBytes []byte, validUntilMS int64,
	cmixParamsJSON []byte) ([]byte, error) {
	// Unmarshal channel ID and parameters
	channelID, params, err := parseChannelsParameters(
		channelIdBytes, cmixParamsJSON)
	if err != nil {
		return nil, err
	}

	// Calculate lease
	lease := time.Duration(validUntilMS) * time.Millisecond
	if validUntilMS == ValidForeverBindings {
		lease = channels.ValidForever
	}

	// Send invite
	messageID, rnd, ephID, err := cm.api.SendSilent(
		channelID, lease, params.CMIX)

	// Construct send report
	return constructChannelSendReport(&messageID, rnd.ID, &ephID)

}

////////////////////////////////////////////////////////////////////////////////
// Admin Sending                                                              //
////////////////////////////////////////////////////////////////////////////////

// SendAdminGeneric is used to send a raw message over a channel encrypted with
// admin keys, identifying it as sent by the admin. In general, it should be
// wrapped in a function that defines the wire protocol.
//
// If the final message, before being sent over the wire, is too long, this will
// return an error. The message must be at most 510 bytes long.
//
// If the user is not an admin of the channel (i.e. does not have a private key
// for the channel saved to storage), then the error [channels.NotAnAdminErr] is
// returned.
//
// Parameters:
//   - channelIdBytes - Marshalled bytes of the channel's [id.ID].
//   - messageType - The message type of the message. This will be a valid
//     [channels.MessageType].
//   - message - The contents of the message. The message should be at most 510
//     bytes. This need not be of data type string, as the message could be a
//     specified format that the channel may recognize.
//   - validUntilMS - The lease of the message. This will be how long the
//     message is available from the network, in milliseconds. As per the
//     [channels.Manager] documentation, this has different meanings depending
//     on the use case. These use cases may be generic enough that they will not
//     be enumerated here. Use [channels.ValidForever] to last the max message
//     life.
//   - tracked - Set tracked to true if the message should be tracked in the
//     sendTracker, which allows messages to be shown locally before they are
//     received on the network. In general, all messages that will be displayed
//     to the user should be tracked while all actions should not be.
//   - cmixParamsJSON - A JSON marshalled [xxdk.CMIXParams]. This may be empty,
//     and [GetDefaultCMixParams] will be used internally.
//
// Returns:
//   - []byte - JSON of [ChannelSendReport].
func (cm *ChannelsManager) SendAdminGeneric(channelIdBytes []byte,
	messageType int, message []byte, validUntilMS int64, tracked bool,
	cmixParamsJSON []byte) ([]byte, error) {
	// Unmarshal channel ID and parameters
	channelID, params, err :=
		parseChannelsParameters(channelIdBytes, cmixParamsJSON)
	if err != nil {
		return nil, err
	}

	msgType := channels.MessageType(messageType)

	// Calculate lease
	lease := time.Duration(validUntilMS) * time.Millisecond
	if validUntilMS == ValidForeverBindings {
		lease = channels.ValidForever
	}

	// Send admin message
	messageID, rnd, ephID, err := cm.api.SendAdminGeneric(
		channelID, msgType, message, lease, tracked, params.CMIX)
	if err != nil {
		return nil, err
	}

	// Construct send report
	return constructChannelSendReport(&messageID, rnd.ID, &ephID)
}

// DeleteMessage deletes the targeted message from user's view. Users may delete
// their own messages but only the channel admin can delete other user's
// messages. If the user is not an admin of the channel or if they are not the
// sender of the targetMessage, then the error [channels.NotAnAdminErr] is
// returned.
//
// If undoAction is true, then the targeted message is un-deleted.
//
// Clients will drop the deletion if they do not recognize the target
// message.
//
// Parameters:
//   - channelIdBytes - Marshalled bytes of channel [id.ID].
//   - targetMessageIdBytes - The marshalled [channel.MessageID] of the message
//     you want to delete.
//   - cmixParamsJSON - JSON of [xxdk.CMIXParams]. This may be empty, and
//     [GetDefaultCMixParams] will be used internally.
//
// Returns:
//   - []byte - JSON of [ChannelSendReport].
func (cm *ChannelsManager) DeleteMessage(channelIdBytes,
	targetMessageIdBytes, cmixParamsJSON []byte) ([]byte, error) {

	// Unmarshal channel ID and parameters
	channelID, params, err :=
		parseChannelsParameters(channelIdBytes, cmixParamsJSON)
	if err != nil {
		return nil, err
	}

	// Unmarshal message ID
	targetedMessageID := cryptoMessage.ID{}
	copy(targetedMessageID[:], targetMessageIdBytes)

	// Send message deletion
	messageID, rnd, ephID, err :=
		cm.api.DeleteMessage(channelID, targetedMessageID, params.CMIX)
	if err != nil {
		return nil, err
	}

	// Construct send report
	return constructChannelSendReport(&messageID, rnd.ID, &ephID)
}

// PinMessage pins the target message to the top of a channel view for all users
// in the specified channel. Only the channel admin can pin user messages; if
// the user is not an admin of the channel, then the error
// [channels.NotAnAdminErr] is returned.
//
// If undoAction is true, then the targeted message is unpinned.
//
// Clients will drop the pin if they do not recognize the target message.
//
// Parameters:
//   - channelIdBytes - Marshalled bytes of channel [id.ID].
//   - targetMessageIdBytes - The marshalled [channel.MessageID] of the message
//     you want to pin.
//   - undoAction - Set to true to unpin the message.
//   - validUntilMS - The time, in milliseconds, that the message should be
//     pinned. To remain pinned indefinitely, use [ValidForever].
//   - cmixParamsJSON - JSON of [xxdk.CMIXParams]. This may be empty, and
//     [GetDefaultCMixParams] will be used internally.
//
// Returns:
//   - []byte - JSON of [ChannelSendReport].
func (cm *ChannelsManager) PinMessage(channelIdBytes,
	targetMessageIdBytes []byte, undoAction bool, validUntilMS int,
	cmixParamsJSON []byte) ([]byte, error) {

	// Unmarshal channel ID and parameters
	channelID, params, err :=
		parseChannelsParameters(channelIdBytes, cmixParamsJSON)
	if err != nil {
		return nil, err
	}

	// Unmarshal message ID
	targetedMessageID := cryptoMessage.ID{}
	copy(targetedMessageID[:], targetMessageIdBytes)

	// Calculate lease
	validUntil := time.Duration(validUntilMS) * time.Millisecond
	if validUntilMS == ValidForeverBindings {
		validUntil = channels.ValidForever
	}

	// Send message pin
	messageID, rnd, ephID, err := cm.api.PinMessage(
		channelID, targetedMessageID, undoAction, validUntil, params.CMIX)
	if err != nil {
		return nil, err
	}

	// Construct send report
	return constructChannelSendReport(&messageID, rnd.ID, &ephID)
}

// MuteUser is used to mute a user in a channel. Muting a user will cause all
// future messages from the user being dropped on reception. Muted users are
// also unable to send messages. Only the channel admin can mute a user; if the
// user is not an admin of the channel, then the error [channels.NotAnAdminErr]
// is returned.
//
// If undoAction is true, then the targeted user will be unmuted.
//
// Parameters:
//   - channelIdBytes - Marshalled bytes of channel [id.ID].
//   - mutedUserPubKeyBytes - The [ed25519.PublicKey] of the user you want to
//     mute.
//   - undoAction - Set to true to unmute the user.
//   - validUntilMS - The time, in milliseconds, that the user should be muted.
//     To remain muted indefinitely, use [ValidForever].
//   - cmixParamsJSON - JSON of [xxdk.CMIXParams]. This may be empty, and
//     [GetDefaultCMixParams] will be used internally.
//
// Returns:
//   - []byte - JSON of [ChannelSendReport].
func (cm *ChannelsManager) MuteUser(channelIdBytes, mutedUserPubKeyBytes []byte,
	undoAction bool, validUntilMS int, cmixParamsJSON []byte) ([]byte, error) {
	// Unmarshal channel ID and parameters
	channelID, params, err :=
		parseChannelsParameters(channelIdBytes, cmixParamsJSON)
	if err != nil {
		return nil, err
	}

	// Unmarshal Ed25519 public key
	if len(mutedUserPubKeyBytes) != ed25519.PublicKeySize {
		return nil, errors.Errorf(
			"user ED25519 public key must be %d bytes, received %d bytes",
			ed25519.PublicKeySize, len(mutedUserPubKeyBytes))
	}

	// Calculate lease
	validUntil := time.Duration(validUntilMS) * time.Millisecond
	if validUntilMS == ValidForeverBindings {
		validUntil = channels.ValidForever
	}

	// Send message to mute user
	messageID, rnd, ephID, err := cm.api.MuteUser(
		channelID, mutedUserPubKeyBytes, undoAction, validUntil, params.CMIX)
	if err != nil {
		return nil, err
	}

	// Construct send report
	return constructChannelSendReport(&messageID, rnd.ID, &ephID)
}

// parseChannelsParameters is a helper function for the Send functions. It
// parses the channel ID and the passed in parameters into their respective
// objects. These objects are passed into the API via the internal send
// functions.
func parseChannelsParameters(channelIdBytes, cmixParamsJSON []byte) (
	*id.ID, xxdk.CMIXParams, error) {
	// Unmarshal channel ID
	channelID, err := id.Unmarshal(channelIdBytes)
	if err != nil {
		return nil, xxdk.CMIXParams{}, err
	}

	// Unmarshal cmix params
	params, err := parseCMixParams(cmixParamsJSON)
	if err != nil {
		return nil, xxdk.CMIXParams{}, err
	}

	return channelID, params, nil
}

// constructChannelSendReport is a helper function that returns a JSON
// marshalled ChannelSendReport containing the message ID, round ID, and
// ephemeral ID. If messageID or ephID are nil, they are excluded from the JSON.
func constructChannelSendReport(messageID *cryptoMessage.ID,
	roundID id.Round, ephID *ephemeral.Id) ([]byte, error) {
	// Construct send report
	chanSendReport := ChannelSendReport{
		MessageID:  messageID.Marshal(),
		RoundsList: makeRoundsList(roundID),
		EphId:      ephID.Int64(),
	}

	if messageID != nil {
		chanSendReport.MessageID = messageID.Marshal()
	}
	if ephID != nil {
		chanSendReport.EphId = ephID.Int64()
	}

	// Marshal send report
	return json.Marshal(chanSendReport)
}

////////////////////////////////////////////////////////////////////////////////
// Other Channel Actions                                                      //
////////////////////////////////////////////////////////////////////////////////

// GetIdentity returns the public identity ([channel.Identity]) of the user
// associated with this channel manager.
//
// Returns:
//   - []byte - JSON of [channel.Identity].
func (cm *ChannelsManager) GetIdentity() ([]byte, error) {
	i := cm.api.GetIdentity()
	return json.Marshal(&i)
}

// ExportPrivateIdentity encrypts the private identity using the password and
// exports it to a portable string.
//
// Parameters:
//   - password - The password used to encrypt the private identity.
//
// Returns:
//   - []byte - Encrypted portable private identity.
func (cm *ChannelsManager) ExportPrivateIdentity(password string) ([]byte, error) {
	return cm.api.ExportPrivateIdentity(password)
}

// GetStorageTag returns the tag where this manager is stored. To be used when
// loading the manager. The storage tag is derived from the public key.
func (cm *ChannelsManager) GetStorageTag() string {
	return cm.api.GetStorageTag()
}

// SetNickname sets the nickname for a given channel. The nickname must be valid
// according to [IsNicknameValid].
//
// Parameters:
//   - nickname - The new nickname.
//   - channelIDBytes - The marshalled bytes of the channel's [id.ID].
func (cm *ChannelsManager) SetNickname(
	nickname string, channelIDBytes []byte) error {
	channelID, err := id.Unmarshal(channelIDBytes)
	if err != nil {
		return err
	}
	return cm.api.SetNickname(nickname, channelID)
}

// DeleteNickname removes the nickname for a given channel. The name will revert
// back to the codename for this channel instead.
//
// Parameters:
//   - channelIDBytes - The marshalled bytes of the channel's [id.ID].
func (cm *ChannelsManager) DeleteNickname(channelIDBytes []byte) error {
	channelID, err := id.Unmarshal(channelIDBytes)
	if err != nil {
		return err
	}
	return cm.api.DeleteNickname(channelID)
}

// GetNickname returns the nickname set for a given channel.
//
// Parameters:
//   - channelIDBytes - The marshalled bytes of the channel's [id.ID].
//
// Returns:
//   - string - The nickname for the channel.
func (cm *ChannelsManager) GetNickname(channelIDBytes []byte) (string, error) {
	channelID, err := id.Unmarshal(channelIDBytes)
	if err != nil {
		return "", err
	}
	nickname, _ := cm.api.GetNickname(channelID)
	return nickname, nil
}

// IsNicknameValid checks if a nickname is valid.
//
// Rules:
//  1. A nickname must not be longer than 24 characters.
//  2. A nickname must not be shorter than 1 character.
//
// Parameters:
//   - nickname - Nickname to check.
func IsNicknameValid(nickname string) error {
	return channels.IsNicknameValid(nickname)
}

// Muted returns true if the user is currently muted in the given channel.
//
// Parameters:
//   - channelIDBytes - The marshalled bytes of the channel's [id.ID].
//
// Returns:
//   - bool - True if the user is muted in the channel and false otherwise.
func (cm *ChannelsManager) Muted(channelIDBytes []byte) (bool, error) {
	channelID, err := id.Unmarshal(channelIDBytes)
	if err != nil {
		return false, err
	}
	return cm.api.Muted(channelID), nil
}

// GetMutedUsers returns the list of the public keys for each muted user in
// the channel. If there are no muted user or if the channel does not exist,
// an empty list is returned.
//
// Parameters:
//   - channelIDBytes - The marshalled bytes of the channel's [id.ID].
//
// Returns:
//   - []byte - JSON of an array of ed25519.PublicKey. Look below for an
//     example.
//
// Example return:
//
//	["k2IrybDXjJtqxjS6Tx/6m3bXvT/4zFYOJnACNWTvESE=","ocELv7KyeCskLz4cm0klLWhmFLYvQL2FMDco79GTXYw=","mmxoDgoTEYwaRyEzq5Npa24IIs+3B5LXhll/8K5yCv0="]
func (cm *ChannelsManager) GetMutedUsers(channelIDBytes []byte) ([]byte, error) {
	channelID, err := id.Unmarshal(channelIDBytes)
	if err != nil {
		return nil, err
	}

	return json.Marshal(cm.api.GetMutedUsers(channelID))
}

////////////////////////////////////////////////////////////////////////////////
// Notifications                                                              //
////////////////////////////////////////////////////////////////////////////////

// GetNotificationLevel returns the [channels.NotificationLevel] for the given
// channel.
//
// Parameters:
//   - channelIDBytes - The marshalled bytes of the channel's [id.ID].
//
// Returns:
//   - int - The [channels.NotificationLevel] for the channel.
func (cm *ChannelsManager) GetNotificationLevel(
	channelIDBytes []byte) (int, error) {
	channelID, err := id.Unmarshal(channelIDBytes)
	if err != nil {
		return 0, err
	}

	level, err := cm.api.GetNotificationLevel(channelID)
	return int(level), err
}

// SetMobileNotificationsLevel sets the notification level for the given
// channel. The [channels.NotificationLevel] dictates the type of notifications
// received and the status controls weather the notification is push or in-app.
// If muted, both the level and status must be set to mute.
//
// To use push notifications, a token must be registered with the notification
// manager. Note, when enabling push notifications, information may be shared
// with third parties (i.e., Firebase and Google's Palantir) and may represent a
// security risk to the user.
//
// Parameters:
//   - channelIDBytes - The marshaled bytes of the channel's [id.ID].
//   - level - The [channels.NotificationLevel] to set for the channel.
//   - status - The [notifications.NotificationState] to set for the channel.
func (cm *ChannelsManager) SetMobileNotificationsLevel(
	channelIDBytes []byte, level, status int) error {
	channelID, err := id.Unmarshal(channelIDBytes)
	if err != nil {
		return err
	}

	return cm.api.SetMobileNotificationsLevel(
		channelID, channels.NotificationLevel(level),
		clientNotif.NotificationState(status))
}

// GetNotificationReportsForMe checks the notification data against the filter
// list to determine which notifications belong to the user. A list of
// notification reports is returned detailing all notifications for the user.
//
// Parameters:
//   - notificationFilterJSON - JSON of a slice of [channels.NotificationFilter].
//   - notificationDataJSON - JSON of a slice of [notifications.Data].
//
// Example JSON of a slice of [channels.NotificationFilter]:
// [
//
//	  {
//	    "identifier": "O8NUg0KaDo18ybTKajXM/sgqEYS37+lewPhGV/2sMAUDYXN5bUlkZW50aWZpZXI=",
//	    "channelID": "O8NUg0KaDo18ybTKajXM/sgqEYS37+lewPhGV/2sMAUD",
//	    "tags": ["6de69009a93d53793ee344e8fb48fae194eaf51861d3cc51c7348c337d13aedf-usrping"],
//	    "allowLists": {
//	      "allowWithTags": {},
//	      "allowWithoutTags": {"102":{}, "2":{}}
//	    }
//	  },
//	  {
//	    "identifier": "O8NUg0KaDo18ybTKajXM/sgqEYS37+lewPhGV/2sMAUDc3ltSWRlbnRpZmllcg==",
//	    "channelID": "O8NUg0KaDo18ybTKajXM/sgqEYS37+lewPhGV/2sMAUD",
//	    "tags": ["6de69009a93d53793ee344e8fb48fae194eaf51861d3cc51c7348c337d13aedf-usrping"],
//	    "allowLists": {
//	      "allowWithTags": {},
//	      "allowWithoutTags": {"1":{}, "40000":{}}
//	    }
//	  },
//	  {
//	    "identifier": "jCRgFRQvzzKOb8DJ0fqCRLgr9kiHN9LpqHXVhyHhhlQDYXN5bUlkZW50aWZpZXI=",
//	    "channelID": "jCRgFRQvzzKOb8DJ0fqCRLgr9kiHN9LpqHXVhyHhhlQD",
//	    "tags": ["6de69009a93d53793ee344e8fb48fae194eaf51861d3cc51c7348c337d13aedf-usrping"],
//	    "allowLists": {
//	      "allowWithTags": {},
//	      "allowWithoutTags": {"102":{}, "2":{}}
//	    }
//	  }
//	]
//
// Example JSON of a slice of [notifications.Data]:
//
//	[
//	  {
//	    "EphemeralID": -6475,
//	    "RoundID": 875,
//	    "IdentityFP": "jWG/UuxRjD80HEo0WX3KYIag5LCfgaWKAg==",
//	    "MessageHash": "hDGE46QWa3d70y5nJTLbEaVmrFJHOyp2"
//	  },
//	  {
//	    "EphemeralID": -2563,
//	    "RoundID": 875,
//	    "IdentityFP": "gL4nhCGKPNBm6YZ7KC0v4JThw65N9bRLTQ==",
//	    "MessageHash": "WcS4vGrSWDK8Kj7JYOkMo8kSh1Xti94V"
//	  },
//	  {
//	    "EphemeralID": -13247,
//	    "RoundID": 875,
//	    "IdentityFP": "qV3uD++VWPhD2rRMmvrP9j8hp+jpFSsUHg==",
//	    "MessageHash": "VX6Tw7N48j7U2rRXYle20mFZi0If4CB1"
//	  }
//	]
//
// Returns:
//   - []byte - JSON of a slice of [channels.NotificationReport].
func GetNotificationReportsForMe(notificationFilterJSON,
	notificationDataJSON []byte) ([]byte, error) {
	var nfs []channels.NotificationFilter
	if err := json.Unmarshal(notificationFilterJSON, &nfs); err != nil {
		return nil, err
	}

	var notifData []*notifications.Data
	if err := json.Unmarshal(notificationDataJSON, &notifData); err != nil {
		return nil, err
	}

	nrs := channels.GetNotificationReportsForMe(nfs, notifData)

	return json.Marshal(nrs)
}

////////////////////////////////////////////////////////////////////////////////
// Admin Management                                                           //
////////////////////////////////////////////////////////////////////////////////

// IsChannelAdmin returns true if the user is an admin of the channel.
//
// Parameters:
//   - channelIDBytes - The marshalled bytes of the channel's [id.ID].
//
// Returns:
//   - bool - True if the user is an admin in the channel and false otherwise.
func (cm *ChannelsManager) IsChannelAdmin(channelIDBytes []byte) (bool, error) {
	channelID, err := id.Unmarshal(channelIDBytes)
	if err != nil {
		return false, err
	}
	return cm.api.IsChannelAdmin(channelID), nil
}

// ExportChannelAdminKey gets the private key for the given channel ID, encrypts
// it with the provided encryptionPassword, and exports it into a portable
// format. Returns an error if the user is not an admin of the channel.
//
// This key can be provided to other users in a channel to grant them admin
// access using [ChannelsManager.ImportChannelAdminKey].
//
// The private key is encrypted using a key generated from the password using
// Argon2. Each call to ExportChannelAdminKey produces a different encrypted
// packet regardless if the same password is used for the same channel. It
// cannot be determined which channel the payload is for nor that two payloads
// are for the same channel.
//
// The passwords between each call are not related. They can be the same or
// different with no adverse impact on the security properties.
//
// Parameters:
//   - channelIdBytes - Marshalled bytes of the channel's [id.ID].
//   - encryptionPassword - The password used to encrypt the private key. The
//     passwords between each call are not related. They can be the same or
//     different with no adverse impact on the security properties.
//
// Returns:
//   - Portable string of the channel private key encrypted with the password.
func (cm *ChannelsManager) ExportChannelAdminKey(
	channelIDBytes []byte, encryptionPassword string) ([]byte, error) {
	channelID, err := id.Unmarshal(channelIDBytes)
	if err != nil {
		return nil, err
	}

	return cm.api.ExportChannelAdminKey(channelID, encryptionPassword)
}

// VerifyChannelAdminKey verifies that the encrypted private key can be
// decrypted and that it matches the expected channel. Returns false if private
// key does not belong to the given channel.
//
// Parameters:
//   - channelIdBytes - Marshalled bytes of the channel's [id.ID].
//   - encryptionPassword - The password used to encrypt the private key.
//   - encryptedPrivKey - The encrypted channel private key packet.
//
// Returns:
//   - bool - True if the private key belongs to the channel and false
//     otherwise.
//   - Returns the error [channels.WrongPasswordErr] for an invalid password.
//   - Returns the error [channels.ChannelDoesNotExistsErr] if the channel has
//     not already been joined.
func (cm *ChannelsManager) VerifyChannelAdminKey(
	channelIdBytes []byte, encryptionPassword string, encryptedPrivKey []byte) (
	bool, error) {
	channelID, err := id.Unmarshal(channelIdBytes)
	if err != nil {
		return false, err
	}

	return cm.api.VerifyChannelAdminKey(
		channelID, encryptionPassword, encryptedPrivKey)
}

// ImportChannelAdminKey decrypts and imports the given encrypted private key
// and grants the user admin access to the channel the private key belongs to.
// Returns an error if the private key cannot be decrypted or if the private key
// is for the wrong channel.
//
// Parameters:
//   - channelIdBytes - Marshalled bytes of the channel's [id.ID].
//   - encryptionPassword - The password used to encrypt the private key.
//   - encryptedPrivKey - The encrypted channel private key packet.
//
// Returns:
//   - Returns the error [channels.WrongPasswordErr] for an invalid password.
//   - Returns the error [channels.ChannelDoesNotExistsErr] if the channel has
//     not already been joined.
//   - Returns the error [channels.WrongPrivateKeyErr] if the private key does
//     not belong to the channel.
func (cm *ChannelsManager) ImportChannelAdminKey(channelIdBytes []byte,
	encryptionPassword string, encryptedPrivKey []byte) error {
	channelID, err := id.Unmarshal(channelIdBytes)
	if err != nil {
		return err
	}

	return cm.api.ImportChannelAdminKey(
		channelID, encryptionPassword, encryptedPrivKey)
}

// DeleteChannelAdminKey deletes the private key for the given channel.
//
// CAUTION: This will remove admin access. This cannot be undone. If the
// private key is deleted, it cannot be recovered and the channel can never
// have another admin.
//
// Parameters:
//   - channelIdBytes - Marshalled bytes of the channel's [id.ID].
func (cm *ChannelsManager) DeleteChannelAdminKey(channelIdBytes []byte) error {
	channelID, err := id.Unmarshal(channelIdBytes)
	if err != nil {
		return err
	}

	return cm.api.DeleteChannelAdminKey(channelID)
}

////////////////////////////////////////////////////////////////////////////////
// Channel Receiving Logic and Callback Registration                          //
////////////////////////////////////////////////////////////////////////////////

// ReceivedChannelMessageReport is a report structure returned via the
// [ChannelMessageReceptionCallback]. This report gives the context for the
// channel the message was sent to and the message itself. This is returned via
// the callback as JSON marshalled bytes.
//
// JSON Example:
//
//	{
//	  "ChannelId": "AQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
//	  "MessageId": "3S6DiVjWH9mLmjy1oaam/3x45bJQzOW6u2KgeUn59wA=",
//	  "ReplyTo":"cxMyGUFJ+Ff1Xp2X+XkIpOnNAQEZmv8SNP5eYH4tCik=",
//	  "MessageType": 42,
//	  "SenderUsername": "hunter2",
//	  "Content": "YmFuX2JhZFVTZXI=",
//	  "Timestamp": 1662502150335283000,
//	  "Lease": 25,
//	  "Rounds": [ 1, 4, 9],
//	}
type ReceivedChannelMessageReport struct {
	ChannelId   []byte
	MessageId   []byte
	MessageType int
	Nickname    string
	PubKey      []byte
	Codeset     int
	Content     []byte
	Timestamp   int64
	Lease       int64
	RoundsList
}

// ChannelMessageReceptionCallback is the callback that returns the context for
// a channel message via the Callback.
//
// It must return a unique UUID for the message by which it can be referenced
// later.
type ChannelMessageReceptionCallback interface {
	Callback(receivedChannelMessageReport []byte, err error) int
}

// RegisterReceiveHandler registers a listener for non-default message types so
// that they can be processed by modules. It is important that such modules
// collective up with the event model implementation.
//
// There can only be one handler per [channels.MessageType]; the error
// [channels.MessageTypeAlreadyRegistered] will be returned on multiple
// registrations of the same type.
//
// Parameters:
//   - messageType - The [channels.MessageType] that the listener listens for.
//   - listenerCb - The callback which will be executed when a channel message
//     of messageType is received.
//   - name - A name describing what type of messages the listener picks up.
//     This is used for debugging and logging.
//   - userSpace - Set to true if this listener can receive messages from normal
//     users.
//   - adminSpace - Set to true if this listener can receive messages from
//     admins.
//   - mutedSpace - Set to true if this listener can receive messages from muted
//     users.
func (cm *ChannelsManager) RegisterReceiveHandler(messageType int,
	listenerCb ChannelMessageReceptionCallback, name string, userSpace,
	adminSpace, mutedSpace bool) error {

	// Wrap callback around backend interface
	cb := channels.MessageTypeReceiveMessage(
		func(channelID *id.ID, messageID cryptoMessage.ID,
			messageType channels.MessageType, nickname string, content,
			encryptedPayload []byte, pubKey ed25519.PublicKey, dmToken uint32,
			codeset uint8, timestamp, originatingTimestamp time.Time,
			lease time.Duration, originatingRound id.Round, round rounds.Round,
			status channels.SentStatus, fromAdmin, hidden bool) uint64 {
			rcm := ReceivedChannelMessageReport{
				ChannelId:   channelID.Marshal(),
				MessageId:   messageID.Marshal(),
				MessageType: int(messageType),
				Nickname:    nickname,
				PubKey:      pubKey,
				Codeset:     int(codeset),
				Content:     content,
				Timestamp:   timestamp.UnixNano(),
				Lease:       int64(lease),
				RoundsList:  makeRoundsList(round.ID),
			}

			return uint64(listenerCb.Callback(json.Marshal(rcm)))
		})

	// Register handler
	return cm.api.RegisterReceiveHandler(channels.MessageType(messageType),
		channels.NewReceiveMessageHandler(
			name, cb, userSpace, adminSpace, mutedSpace))
}

////////////////////////////////////////////////////////////////////////////////
// Event Model Logic                                                          //
////////////////////////////////////////////////////////////////////////////////

// EventModelBuilder builds an event model
type EventModelBuilder interface {
	Build(path string) EventModel
}

// EventModel is an interface which an external party which uses the channels
// system passed an object which adheres to in order to get events on the
// channel.
type EventModel interface {
	// JoinChannel is called whenever a channel is joined locally.
	//
	// Parameters:
	//  - channel - Returns the pretty print representation of a channel.
	JoinChannel(channel string)

	// LeaveChannel is called whenever a channel is left locally.
	//
	// Parameters:
	//  - ChannelID - The marshalled channel [id.ID].
	LeaveChannel(channelID []byte)

	// ReceiveMessage is called whenever a message is received on a given
	// channel. It may be called multiple times on the same message. It is
	// incumbent on the user of the API to filter such called by message ID.
	//
	// The API needs to return a UUID of the message that can be referenced at a
	// later time.
	//
	// messageID, timestamp, and roundID are all nillable and may be updated
	// based upon the UUID at a later date. A value of 0 will be passed for a
	// nilled timestamp or roundID.
	//
	// nickname may be empty, in which case the UI is expected to display the
	// codename.
	//
	// messageType is included in the call; it will always be [channels.Text]
	// (1) for this call, but it may be required in downstream databases.
	//
	// Parameters:
	//  - channelID - The marshalled channel [id.ID].
	//  - messageID - The bytes of the [channel.MessageID] of the received
	//    message.
	//  - nickname - The nickname of the sender of the message.
	//  - text - The content of the message.
	//  - timestamp - Time the message was received; represented as nanoseconds
	//    since unix epoch.
	//  - pubKey - The sender's Ed25519 public key.
	//  - codeset - The codeset version.
	//  - lease - The number of nanoseconds that the message is valid for.
	//  - roundID - The ID of the round that the message was received on.
	//  - messageType - the type of the message, always 1 for this call
	//  - status - the [channels.SentStatus] of the message.
	//
	// Statuses will be enumerated as such:
	//  Sent      =  0
	//  Delivered =  1
	//  Failed    =  2
	//
	// Returns:
	//  - int64 - A non-negative unique UUID for the message that it can be
	//    referenced by later with [EventModel.UpdateFromUUID].
	ReceiveMessage(channelID, messageID []byte, nickname, text string,
		pubKey []byte, dmToken int32, codeset int, timestamp, lease, roundID,
		messageType, status int64, hidden bool) int64

	// ReceiveReply is called whenever a message is received that is a reply on
	// a given channel. It may be called multiple times on the same message. It
	// is incumbent on the user of the API to filter such called by message ID.
	//
	// Messages may arrive our of order, so a reply, in theory, can arrive
	// before the initial message. As a result, it may be important to buffer
	// replies.
	//
	// The API needs to return a UUID of the message that can be referenced at a
	// later time.
	//
	// messageID, timestamp, and roundID are all nillable and may be updated
	// based upon the UUID at a later date. A value of 0 will be passed for a
	// nilled timestamp or roundID.
	//
	// nickname may be empty, in which case the UI is expected to display the
	// codename.
	//
	// messageType type is included in the call; it will always be
	// [channels.Text] (1) for this call, but it may be required in downstream
	// databases.
	//
	// Parameters:
	//  - channelID - The marshalled channel [id.ID].
	//  - messageID - The bytes of the [channel.MessageID] of the received
	//    message.
	//  - reactionTo - The [channel.MessageID] for the message that received a
	//    reply.
	//  - nickname - The nickname of the sender of the message.
	//  - text - The content of the message.
	//  - pubKey - The sender's Ed25519 public key.
	//  - codeset - The codeset version.
	//  - timestamp - Time the message was received; represented as nanoseconds
	//    since unix epoch.
	//  - lease - The number of nanoseconds that the message is valid for.
	//  - roundID - The ID of the round that the message was received on.
	//  - messageType - the type of the message, always 1 for this call
	//  - status - the [channels.SentStatus] of the message.
	//
	// Statuses will be enumerated as such:
	//  Sent      =  0
	//  Delivered =  1
	//  Failed    =  2
	//
	// Returns:
	//  - int64 - A non-negative unique UUID for the message that it can be
	//    referenced by later with [EventModel.UpdateFromUUID].
	ReceiveReply(channelID, messageID, reactionTo []byte, nickname, text string,
		pubKey []byte, dmToken int32, codeset int, timestamp, lease, roundID,
		messageType, status int64, hidden bool) int64

	// ReceiveReaction is called whenever a reaction to a message is received on
	// a given channel. It may be called multiple times on the same reaction. It
	// is incumbent on the user of the API to filter such called by message ID.
	//
	// Messages may arrive our of order, so a reply, in theory, can arrive
	// before the initial message. As a result, it may be important to buffer
	// replies.
	//
	// The API needs to return a UUID of the message that can be referenced at a
	// later time.
	//
	// messageID, timestamp, and roundID are all nillable and may be updated
	// based upon the UUID at a later date. A value of 0 will be passed for a
	// nilled timestamp or roundID.
	//
	// nickname may be empty, in which case the UI is expected to display the
	// codename.
	//
	// messageType type is included in the call; it will always be
	// [channels.Text] (1) for this call, but it may be required in downstream
	// databases.
	//
	// Parameters:
	//  - channelID - The marshalled channel [id.ID].
	//  - messageID - The bytes of the [channel.MessageID] of the received
	//    message.
	//  - reactionTo - The [channel.MessageID] for the message that received a
	//    reply.
	//  - nickname - The nickname of the sender of the message.
	//  - reaction - The contents of the reaction message.
	//  - pubKey - The sender's Ed25519 public key.
	//  - codeset - The codeset version.
	//  - timestamp - Time the message was received; represented as nanoseconds
	//    since unix epoch.
	//  - lease - The number of nanoseconds that the message is valid for.
	//  - roundID - The ID of the round that the message was received on.
	//  - messageType - the type of the message, always 1 for this call
	//  - status - the [channels.SentStatus] of the message.
	//
	// Statuses will be enumerated as such:
	//  Sent      =  0
	//  Delivered =  1
	//  Failed    =  2
	//
	// Returns:
	//  - int64 - A non-negative unique UUID for the message that it can be
	//    referenced by later with [EventModel.UpdateFromUUID].
	ReceiveReaction(channelID, messageID, reactionTo []byte, nickname,
		reaction string, pubKey []byte, dmToken int32, codeset int, timestamp,
		lease, roundID, messageType, status int64, hidden bool) int64

	// UpdateFromUUID is called whenever a message at the UUID is modified.
	//
	// MessageID, Timestamp, RoundID, Pinned, Hidden, and Status in the
	// [MessageUpdateInfo] may be empty (as indicated by their associated
	// boolean) and updated based upon the UUID at a later date.
	//
	// Parameters:
	//  - uuid - The unique identifier of the message in the database.
	//  - messageUpdateInfoJSON - JSON of [MessageUpdateInfo].
	//
	// Returns:
	//  - Returns an error if the message cannot be updated. It must return the
	//	  error from GetNoMessageErr if the message does not exist.
	UpdateFromUUID(uuid int64, messageUpdateInfoJSON []byte) error

	// UpdateFromMessageID is called whenever a message with the message ID is
	// modified.
	//
	// Timestamp, RoundID, Pinned, Hidden, and Status in the [MessageUpdateInfo]
	// may be empty (as indicated by their associated boolean) and updated based
	// upon the UUID at a later date.
	//
	// Parameters:
	//  - messageID - The bytes of the [channel.MessageID] of the received
	//    message.
	//  - messageUpdateInfoJSON - JSON of [MessageUpdateInfo].
	//
	// Returns:
	//  - int64 - A non-negative unique UUID for the message that it can be
	//    referenced by later with [EventModel.UpdateFromUUID].
	//  - Returns an error if the message cannot be updated. It must return the
	//	  error from GetNoMessageErr if the message does not exist.
	UpdateFromMessageID(
		messageID []byte, messageUpdateInfoJSON []byte) (int64, error)

	// GetMessage returns the message with the given [channel.MessageID].
	//
	// Parameters:
	//  - messageID - The bytes of the [channel.MessageID] of the message.
	//
	// Returns:
	//  - JSON of [channels.ModelMessage].
	//  - Returns an error if the message cannot be gotten. It must return the
	//	  error from GetNoMessageErr if the message does not exist.
	GetMessage(messageID []byte) ([]byte, error)

	// DeleteMessage deletes the message with the given [channel.MessageID] from
	// the database.
	//
	// Parameters:
	//  - messageID - The bytes of the [channel.MessageID] of the message.
	//  - Returns an error if the message cannot be deleted. It must return the
	//	  error from GetNoMessageErr if the message does not exist.
	DeleteMessage(messageID []byte) error

	// MuteUser mutes the given user or unmutes them.
	//
	// Parameters:
	//  - channelID - The bytes of the [id.ID] of the channel the user is being
	//    muted in.
	//  - pubKey - The Ed25519 public key of the user that is muted or unmuted.
	MuteUser(channelID, pubkey []byte, unmute bool)
}

// GetNoMessageErr returns the error channels.NoMessageErr, which must be
// returned by EventModel methods (such as EventModel.UpdateFromUUID,
// EventModel.UpdateFromMessageID, and EventModel.GetMessage) when the message
// cannot be found.
func GetNoMessageErr() string {
	return channels.NoMessageErr.Error()
}

// CheckNoMessageErr determines if the error returned by an EventModel function
// indicates that the message or item does not exist. It returns true if the
// error contains channels.NoMessageErr.
func CheckNoMessageErr(err string) bool {
	return channels.CheckNoMessageErr(errors.New(err))
}

// MessageUpdateInfo contains the updated information for a channel message.
// Only update fields that have their set field set as true.
type MessageUpdateInfo struct {
	// MessageID is the bytes of the [channel.MessageID] of the received
	// message.
	MessageID    []byte
	MessageIDSet bool

	// Timestamp, in milliseconds, when the message was sent.
	Timestamp    int64
	TimestampSet bool

	// RoundID is the [id.Round] the message was sent on.
	RoundID    int64
	RoundIDSet bool

	// Pinned is true if the message is pinned.
	Pinned    bool
	PinnedSet bool

	// Hidden is true if the message is hidden
	Hidden    bool
	HiddenSet bool

	// Status is the [channels.SentStatus] of the message.
	//  Sent      =  1
	//  Delivered =  2
	//  Failed    =  3
	Status    int64
	StatusSet bool
}

// toEventModel is a wrapper which wraps an existing channels.EventModel object.
type toEventModel struct {
	em EventModel
}

// NewEventModel is a constructor for a toEventModel. This will take in an
// EventModel and wraps it around the toEventModel.
func NewEventModel(em EventModel) channels.EventModel {
	return &toEventModel{em: em}
}

// JoinChannel is called whenever a channel is joined locally.
func (tem *toEventModel) JoinChannel(channel *cryptoBroadcast.Channel) {
	tem.em.JoinChannel(channel.PrettyPrint())
}

// LeaveChannel is called whenever a channel is left locally.
func (tem *toEventModel) LeaveChannel(channelID *id.ID) {
	tem.em.LeaveChannel(channelID[:])
}

// ReceiveMessage is called whenever a message is received on a given channel.
// It may be called multiple times on the same message. It is incumbent on the
// user of the API to filter such called by message ID.
//
// The API needs to return a UUID of the message that can be referenced at a
// later time.
//
// messageID, timestamp, and round are all nillable and may be updated based
// upon the UUID at a later date. A time of time.Time{} will be passed for a
// nilled timestamp.
//
// nickname may be empty, in which case the UI is expected to display the
// codename.
//
// messageType type is included in the call; it will always be [channels.Text]
// (1) for this call, but it may be required in downstream databases.
func (tem *toEventModel) ReceiveMessage(channelID *id.ID,
	messageID cryptoMessage.ID, nickname, text string, pubKey ed25519.PublicKey,
	dmToken uint32, codeset uint8, timestamp time.Time, lease time.Duration,
	round rounds.Round, messageType channels.MessageType,
	status channels.SentStatus, hidden bool) uint64 {
	return uint64(tem.em.ReceiveMessage(channelID[:], messageID[:], nickname,
		text, pubKey, int32(dmToken), int(codeset), timestamp.UnixNano(),
		int64(lease), int64(round.ID), int64(messageType), int64(status),
		hidden))
}

// ReceiveReply is called whenever a message is received that is a reply on a
// given channel. It may be called multiple times on the same message. It is
// incumbent on the user of the API to filter such called by message ID.
//
// Messages may arrive our of order, so a reply, in theory, can arrive before
// the initial message. As a result, it may be important to buffer replies.
//
// The API needs to return a UUID of the message that can be referenced at a
// later time.
//
// messageID, timestamp, and round are all nillable and may be updated based
// upon the UUID at a later date. A time of time.Time{} will be passed for a
// nilled timestamp.
//
// nickname may be empty, in which case the UI is expected to display the
// codename.
//
// messageType type is included in the call; it will always be [channels.Text]
// (1) for this call, but it may be required in downstream databases.
func (tem *toEventModel) ReceiveReply(channelID *id.ID, messageID,
	reactionTo cryptoMessage.ID, nickname, text string,
	pubKey ed25519.PublicKey, dmToken uint32, codeset uint8,
	timestamp time.Time, lease time.Duration, round rounds.Round,
	messageType channels.MessageType, status channels.SentStatus,
	hidden bool) uint64 {

	return uint64(tem.em.ReceiveReply(channelID[:], messageID[:], reactionTo[:],
		nickname, text, pubKey, int32(dmToken), int(codeset),
		timestamp.UnixNano(), int64(lease), int64(round.ID),
		int64(messageType), int64(status),
		hidden))

}

// ReceiveReaction is called whenever a reaction to a message is received on a
// given channel. It may be called multiple times on the same reaction. It is
// incumbent on the user of the API to filter such called by message ID.
//
// Messages may arrive our of order, so a reply, in theory, can arrive before
// the initial message. As a result, it may be important to buffer replies.
//
// The API needs to return a UUID of the message that can be referenced at a
// later time.
//
// messageID, timestamp, and round are all nillable and may be updated based
// upon the UUID at a later date. A time of time.Time{} will be passed for a
// nilled timestamp.
//
// nickname may be empty, in which case the UI is expected to display the
// codename.
//
// messageType type is included in the call; it will always be [channels.Text]
// (1) for this call, but it may be required in downstream databases.
func (tem *toEventModel) ReceiveReaction(channelID *id.ID, messageID,
	reactionTo cryptoMessage.ID, nickname, reaction string,
	pubKey ed25519.PublicKey, dmToken uint32, codeset uint8,
	timestamp time.Time, lease time.Duration, round rounds.Round,
	messageType channels.MessageType, status channels.SentStatus,
	hidden bool) uint64 {

	return uint64(tem.em.ReceiveReaction(channelID[:], messageID[:],
		reactionTo[:], nickname, reaction, pubKey, int32(dmToken),
		int(codeset), timestamp.UnixNano(), int64(lease),
		int64(round.ID), int64(messageType), int64(status), hidden))
}

// UpdateFromUUID is called whenever a message at the UUID is modified.
//
// messageID, timestamp, round, pinned, hidden, and status are all nillable and
// may be updated based upon the UUID at a later date. If a nil value is passed,
// then make no update.
//
// Returns an error if the message cannot be updated. It must return the error
// from GetNoMessageErr if the message does not exist.
func (tem *toEventModel) UpdateFromUUID(uuid uint64,
	messageID *cryptoMessage.ID, timestamp *time.Time, round *rounds.Round,
	pinned, hidden *bool, status *channels.SentStatus) error {
	var mui MessageUpdateInfo

	if messageID != nil {
		mui.MessageID = messageID.Marshal()
		mui.MessageIDSet = true
	}
	if timestamp != nil {
		mui.Timestamp = timestamp.UnixNano()
		mui.TimestampSet = true
	}
	if round != nil {
		mui.RoundID = int64(round.ID)
		mui.RoundIDSet = true
	}
	if pinned != nil {
		mui.Pinned = *pinned
		mui.PinnedSet = true
	}
	if hidden != nil {
		mui.Hidden = *hidden
		mui.HiddenSet = true
	}
	if status != nil {
		mui.Status = int64(*status)
		mui.StatusSet = true
	}

	muiJSON, err := json.Marshal(mui)
	if err != nil {
		return errors.Errorf(
			"failed to JSON marshal MessageUpdateInfo: %+v", err)
	}

	return tem.em.UpdateFromUUID(int64(uuid), muiJSON)
}

// UpdateFromMessageID is called whenever a message with the message ID is
// modified.
//
// The API needs to return the UUID of the modified message that can be
// referenced at a later time.
//
// timestamp, round, pinned, hidden, and status are all nillable and may be
// updated based upon the UUID at a later date. If a nil value is passed, then
// make no update.
//
// Returns an error if the message cannot be updated. It must return the error
// from GetNoMessageErr if the message does not exist.
func (tem *toEventModel) UpdateFromMessageID(messageID cryptoMessage.ID,
	timestamp *time.Time, round *rounds.Round, pinned, hidden *bool,
	status *channels.SentStatus) (uint64, error) {
	var mui MessageUpdateInfo

	if timestamp != nil {
		mui.Timestamp = timestamp.UnixNano()
		mui.TimestampSet = true
	}
	if round != nil {
		mui.RoundID = int64(round.ID)
		mui.RoundIDSet = true
	}
	if pinned != nil {
		mui.Pinned = *pinned
		mui.PinnedSet = true
	}
	if hidden != nil {
		mui.Hidden = *hidden
		mui.HiddenSet = true
	}
	if status != nil {
		mui.Status = int64(*status)
		mui.StatusSet = true
	}

	muiJSON, err := json.Marshal(mui)
	if err != nil {
		return 0, errors.Errorf(
			"failed to JSON marshal MessageUpdateInfo: %+v", err)
	}

	uuid, err := tem.em.UpdateFromMessageID(messageID.Marshal(), muiJSON)
	return uint64(uuid), err
}

// GetMessage returns the message with the given [channel.MessageID].
//
// It must return the error from GetNoMessageErr if the message does not exist.
//
// Returns an error if the message cannot be gotten. It must return the error
// from GetNoMessageErr if the message does not exist.
func (tem *toEventModel) GetMessage(
	messageID cryptoMessage.ID) (channels.ModelMessage, error) {
	msgJSON, err := tem.em.GetMessage(messageID.Marshal())
	if err != nil {
		return channels.ModelMessage{}, err
	}
	var msg channels.ModelMessage
	return msg, json.Unmarshal(msgJSON, &msg)
}

// DeleteMessage deletes the message with the given [channel.MessageID] from the
// database.
//
// Returns an error if the message cannot be deleted. It must return the error
// from GetNoMessageErr if the message does not exist.
func (tem *toEventModel) DeleteMessage(messageID cryptoMessage.ID) error {
	return tem.em.DeleteMessage(messageID.Marshal())
}

// MuteUser is called when the given user is muted or unmuted.
func (tem *toEventModel) MuteUser(
	channelID *id.ID, pubKey ed25519.PublicKey, unmute bool) {
	tem.em.MuteUser(channelID.Marshal(), pubKey, unmute)
}

////////////////////////////////////////////////////////////////////////////////
// Extension Builder Tracker                                                  //
////////////////////////////////////////////////////////////////////////////////

// channelsExtensionBuilderTrackerSingleton is used to track
// channels.ExtensionBuilder objects so that they can be referenced by ID back
// over the bindings.
var channelsExtensionBuilderTrackerSingleton = &channelsExtensionBuilderTracker{
	tracked: make(map[int]channels.ExtensionBuilder),
	count:   0,
}

// channelsExtensionBuilderTracker keeps track of extant
// channels.ExtensionBuilder objects, preventing race conditions created by
// passing it over the bindings.
type channelsExtensionBuilderTracker struct {
	tracked map[int]channels.ExtensionBuilder
	count   int
	mux     sync.RWMutex
}

// add inserts the channels.ExtensionBuilder into the tracker and returns a
// unique ID it can be referenced by later.
func (ebt *channelsExtensionBuilderTracker) add(
	eb channels.ExtensionBuilder) int {
	ebt.mux.Lock()
	defer ebt.mux.Unlock()

	newID := ebt.count
	ebt.count++

	ebt.tracked[newID] = eb

	return newID
}

// get returns the channels.ExtensionBuilder with the given ID from the tracker
// or false if it does not exist.
func (ebt *channelsExtensionBuilderTracker) get(id int) (
	channels.ExtensionBuilder, bool) {
	ebt.mux.RLock()
	defer ebt.mux.RUnlock()

	eb, exists := ebt.tracked[id]
	return eb, exists
}

// delete removes the channels.ExtensionBuilder with the given ID from the
// tracker.
func (ebt *channelsExtensionBuilderTracker) delete(id int) {
	ebt.mux.Lock()
	defer ebt.mux.Unlock()

	delete(ebt.tracked, id)
}

////////////////////////////////////////////////////////////////////////////////
// Channel ChannelDbCipher                                                    //
////////////////////////////////////////////////////////////////////////////////

// ChannelDbCipher is the bindings layer representation of the [channel.Cipher].
type ChannelDbCipher struct {
	api  cryptoChannel.Cipher
	salt []byte
	id   int
}

// channelDbCipherTrackerSingleton is used to track ChannelDbCipher objects
// so that they can be referenced by ID back over the bindings.
var channelDbCipherTrackerSingleton = &channelDbCipherTracker{
	tracked: make(map[int]*ChannelDbCipher),
	count:   0,
}

// channelDbCipherTracker is a singleton used to keep track of extant
// ChannelDbCipher objects, preventing race conditions created by passing it
// over the bindings.
type channelDbCipherTracker struct {
	tracked map[int]*ChannelDbCipher
	count   int
	mux     sync.RWMutex
}

// create creates a ChannelDbCipher from a [channel.Cipher], assigns it a unique
// ID, and adds it to the channelDbCipherTracker.
func (ct *channelDbCipherTracker) create(c cryptoChannel.Cipher) *ChannelDbCipher {
	ct.mux.Lock()
	defer ct.mux.Unlock()

	chID := ct.count
	ct.count++

	ct.tracked[chID] = &ChannelDbCipher{
		api: c,
		id:  chID,
	}

	return ct.tracked[chID]
}

// get an ChannelDbCipher from the channelDbCipherTracker given its ID.
func (ct *channelDbCipherTracker) get(id int) (*ChannelDbCipher, error) {
	ct.mux.RLock()
	defer ct.mux.RUnlock()

	c, exist := ct.tracked[id]
	if !exist {
		return nil, errors.Errorf(
			"Cannot get ChannelDbCipher for ID %d, does not exist", id)
	}

	return c, nil
}

// delete removes a ChannelDbCipher from the channelDbCipherTracker.
func (ct *channelDbCipherTracker) delete(id int) {
	ct.mux.Lock()
	defer ct.mux.Unlock()

	delete(ct.tracked, id)
}

// GetChannelDbCipherTrackerFromID returns the ChannelDbCipher with the
// corresponding ID in the tracker.
func GetChannelDbCipherTrackerFromID(id int) (*ChannelDbCipher, error) {
	return channelDbCipherTrackerSingleton.get(id)
}

// NewChannelsDatabaseCipher constructs a ChannelDbCipher object.
//
// Parameters:
//   - cmixID - The tracked [Cmix] object ID.
//   - password - The password for storage. This should be the same password
//     passed into [NewCmix].
//   - plaintTextBlockSize - The maximum size of a payload to be encrypted.
//     A payload passed into [ChannelDbCipher.Encrypt] that is larger than
//     plaintTextBlockSize will result in an error.
func NewChannelsDatabaseCipher(cmixID int, password []byte,
	plaintTextBlockSize int) (*ChannelDbCipher, error) {
	// Get user from singleton
	user, err := cmixTrackerSingleton.get(cmixID)
	if err != nil {
		return nil, err
	}

	// Generate RNG
	stream := user.api.GetRng().GetStream()

	// Load or generate a salt
	salt, err := utility.NewOrLoadSalt(
		user.api.GetStorage().GetKV(), stream)
	if err != nil {
		return nil, err
	}

	// Construct a cipher
	c, err := cryptoChannel.NewCipher(
		password, salt, plaintTextBlockSize, stream)
	if err != nil {
		return nil, err
	}

	// Return a cipher
	return channelDbCipherTrackerSingleton.create(c), nil
}

// GetID returns the ID for this ChannelDbCipher in the channelDbCipherTracker.
func (c *ChannelDbCipher) GetID() int {
	return c.id
}

// Encrypt will encrypt the raw data. It will return a ciphertext. Padding is
// done on the plaintext so all encrypted data looks uniform at rest.
//
// Parameters:
//   - plaintext - The data to be encrypted. This must be smaller than the block
//     size passed into [NewChannelsDatabaseCipher]. If it is larger, this will
//     return an error.
func (c *ChannelDbCipher) Encrypt(plaintext []byte) ([]byte, error) {
	return c.api.Encrypt(plaintext)
}

// Decrypt will decrypt the passed in encrypted value. The plaintext will
// be returned by this function. Any padding will be discarded within
// this function.
//
// Parameters:
//   - ciphertext - the encrypted data returned by [ChannelDbCipher.Encrypt].
func (c *ChannelDbCipher) Decrypt(ciphertext []byte) ([]byte, error) {
	return c.api.Decrypt(ciphertext)
}

// MarshalJSON marshals the cipher into valid JSON. This function adheres to the
// json.Marshaler interface.
func (c *ChannelDbCipher) MarshalJSON() ([]byte, error) {
	return c.api.MarshalJSON()
}

// UnmarshalJSON unmarshalls JSON into the cipher. This function adheres to the
// json.Unmarshaler interface.
//
// Note that this function does not transfer the internal RNG. Use
// NewCipherFromJSON to properly reconstruct a cipher from JSON.
func (c *ChannelDbCipher) UnmarshalJSON(data []byte) error {
	return c.api.UnmarshalJSON(data)
}

////////////////////////////////////////////////////////////////////////////////
// UI Callbacks                                                               //
////////////////////////////////////////////////////////////////////////////////

func newChannelUICallbacksWrapper(uicb ChannelUICallbacks) *ChannelUICallbacksWrapper {
	if uicb == nil {
		return nil
	}
	return &ChannelUICallbacksWrapper{Cuic: uicb}
}

// Event Type
const (
	NickNameUpdate int64 = iota
	NotificationUpdate
	MessageReceived
	UserMuted
	MessageDeleted
	AdminKeyUpdate
	DmTokenUpdate
)

// NickNameUpdateJson is describes when your nickname changes due to a change on a
// remote.
//
//	{
//	 "channelID":"KdkEjm+OfQuK4AyZGAqh+XPQaLfRhsO5d2NT1EIScyJX",
//	 "nickname":"billNyeTheScienceGuy",
//	 "exists":true
//	}
type NickNameUpdateJson struct {
	ChannelId *id.ID `json:"channelID"`
	Nickname  string `json:"nickname"`
	Exists    bool   `json:"exists"`
}

// NotificationUpdateJson describes any time a notification
// level changes.
//
// It contains  a slice of [NotificationFilter] for all channels with
// notifications enabled. The [NotificationFilter] is used to determine
// which notifications from the notification server belong to the caller.
//
// It also contains  a map of all channel notification states that have
// changed and all that have been deleted. The maxState is the global state
// set for notifications.
//
// Contains:
//
//   - notificationFilters - JSON of a slice of
//     [channels.NotificationFilter].
//
//   - changedNotificationStates - JSON of a slice of
//     [channels.NotificationState] of added or changed channel notification
//     statuses.
//
//   - deletedNotificationStates - JSON of a slice of [id.ID] of deleted
//     channel notification statuses.
//
//   - maxState - The global notification state.
//
//     {
//     "notificationFilters": [
//     {
//     "identifier": "Z1owNo+GvizWshVW/C5IJ1izPD5oqMkCGr+PsA5If4EDQXN5bW1Ub1B1YmxpY0JjYXN0",
//     "channelID": "Z1owNo+GvizWshVW/C5IJ1izPD5oqMkCGr+PsA5If4ED",
//     "tags": [
//     "af35cdae2159477d79f7ab33bf0bb73ccc1f212bfdc1b3ae78cf398c02878e01-usrping"
//     ],
//     "allowLists": {
//     "allowWithTags": {
//     "2": {}
//     },
//     "allowWithoutTags": {
//     "102": {}
//     }
//     }
//     },
//     {
//     "identifier": "Z1owNo+GvizWshVW/C5IJ1izPD5oqMkCGr+PsA5If4EDU3ltbWV0cmljQnJvYWRjYXN0",
//     "channelID": "Z1owNo+GvizWshVW/C5IJ1izPD5oqMkCGr+PsA5If4ED",
//     "tags": [
//     "af35cdae2159477d79f7ab33bf0bb73ccc1f212bfdc1b3ae78cf398c02878e01-usrping"
//     ],
//     "allowLists": {
//     "allowWithTags": {
//     "1": {},
//     "40000": {}
//     },
//     "allowWithoutTags": {}
//     }
//     },
//     {
//     "identifier": "xsrTzBVFS9s0ccPpgSwBRjCFP5ZYUibswfnhLbjrePoDQXN5bW1Ub1B1YmxpY0JjYXN0",
//     "channelID": "xsrTzBVFS9s0ccPpgSwBRjCFP5ZYUibswfnhLbjrePoD",
//     "tags": [
//     "4f4b35a64a3bd7b06614c2f48d0cdda8b2220ca0fcba78cd2ed11ba38afc92f2-usrping"
//     ],
//     "allowLists": {
//     "allowWithTags": {
//     "2": {}
//     },
//     "allowWithoutTags": {
//     "102": {}
//     }
//     }
//     },
//     {
//     "identifier": "xsrTzBVFS9s0ccPpgSwBRjCFP5ZYUibswfnhLbjrePoDU3ltbWV0cmljQnJvYWRjYXN0",
//     "channelID": "xsrTzBVFS9s0ccPpgSwBRjCFP5ZYUibswfnhLbjrePoD",
//     "tags": [
//     "4f4b35a64a3bd7b06614c2f48d0cdda8b2220ca0fcba78cd2ed11ba38afc92f2-usrping"
//     ],
//     "allowLists": {
//     "allowWithTags": {
//     "1": {},
//     "40000": {}
//     },
//     "allowWithoutTags": {}
//     }
//     },
//     {
//     "identifier": "buqebq3uk/3GeTPKOuzJJXr+rVKfM+TyHed6jFpJkCQDQXN5bW1Ub1B1YmxpY0JjYXN0",
//     "channelID": "buqebq3uk/3GeTPKOuzJJXr+rVKfM+TyHed6jFpJkCQD",
//     "tags": [
//     "72c73b78133739042a5b15c635853d2617324345f611fb272d1fa894a2adf96a-usrping"
//     ],
//     "allowLists": {
//     "allowWithTags": {},
//     "allowWithoutTags": {
//     "102": {},
//     "2": {}
//     }
//     }
//     },
//     {
//     "identifier": "buqebq3uk/3GeTPKOuzJJXr+rVKfM+TyHed6jFpJkCQDU3ltbWV0cmljQnJvYWRjYXN0",
//     "channelID": "buqebq3uk/3GeTPKOuzJJXr+rVKfM+TyHed6jFpJkCQD",
//     "tags": [
//     "72c73b78133739042a5b15c635853d2617324345f611fb272d1fa894a2adf96a-usrping"
//     ],
//     "allowLists": {
//     "allowWithTags": {},
//     "allowWithoutTags": {
//     "1": {},
//     "40000": {}
//     }
//     }
//     },
//     {
//     "identifier": "gZ4uFg/NaSGJVED3hH+PsezGwkZExgPeRxITlfjXZDUDQXN5bW1Ub1B1YmxpY0JjYXN0",
//     "channelID": "gZ4uFg/NaSGJVED3hH+PsezGwkZExgPeRxITlfjXZDUD",
//     "tags": [
//     "0e2aaeacd3cf1d1738cc09f94405b6d1a841af575211cb6f4d39b7d4914d5341-usrping"
//     ],
//     "allowLists": {
//     "allowWithTags": {},
//     "allowWithoutTags": {
//     "102": {},
//     "2": {}
//     }
//     }
//     },
//     {
//     "identifier": "gZ4uFg/NaSGJVED3hH+PsezGwkZExgPeRxITlfjXZDUDU3ltbWV0cmljQnJvYWRjYXN0",
//     "channelID": "gZ4uFg/NaSGJVED3hH+PsezGwkZExgPeRxITlfjXZDUD",
//     "tags": [
//     "0e2aaeacd3cf1d1738cc09f94405b6d1a841af575211cb6f4d39b7d4914d5341-usrping"
//     ],
//     "allowLists": {
//     "allowWithTags": {},
//     "allowWithoutTags": {
//     "1": {},
//     "40000": {}
//     }
//     }
//     },
//     {
//     "identifier": "DZ96YMyBhsNQyC0vACeaaYRYBI4gbzArz7jANLIIR/wDQXN5bW1Ub1B1YmxpY0JjYXN0",
//     "channelID": "DZ96YMyBhsNQyC0vACeaaYRYBI4gbzArz7jANLIIR/wD",
//     "tags": [
//     "cbce1940f0102e791d67412328acfa53f9881494e249eb345efac084224187b6-usrping"
//     ],
//     "allowLists": {
//     "allowWithTags": {
//     "2": {}
//     },
//     "allowWithoutTags": {
//     "102": {}
//     }
//     }
//     },
//     {
//     "identifier": "DZ96YMyBhsNQyC0vACeaaYRYBI4gbzArz7jANLIIR/wDU3ltbWV0cmljQnJvYWRjYXN0",
//     "channelID": "DZ96YMyBhsNQyC0vACeaaYRYBI4gbzArz7jANLIIR/wD",
//     "tags": [
//     "cbce1940f0102e791d67412328acfa53f9881494e249eb345efac084224187b6-usrping"
//     ],
//     "allowLists": {
//     "allowWithTags": {
//     "1": {},
//     "40000": {}
//     },
//     "allowWithoutTags": {}
//     }
//     },
//     {
//     "identifier": "djq86hTU0WjhpbAccQoPKpKpz7K+yxabpvY4iktphqQDQXN5bW1Ub1B1YmxpY0JjYXN0",
//     "channelID": "djq86hTU0WjhpbAccQoPKpKpz7K+yxabpvY4iktphqQD",
//     "tags": [
//     "ae893892e2b1253bb39fdac5f81b08c5ab69cb06ee79345715972c1af6c5125f-usrping"
//     ],
//     "allowLists": {
//     "allowWithTags": {
//     "2": {}
//     },
//     "allowWithoutTags": {
//     "102": {}
//     }
//     }
//     },
//     {
//     "identifier": "djq86hTU0WjhpbAccQoPKpKpz7K+yxabpvY4iktphqQDU3ltbWV0cmljQnJvYWRjYXN0",
//     "channelID": "djq86hTU0WjhpbAccQoPKpKpz7K+yxabpvY4iktphqQD",
//     "tags": [
//     "ae893892e2b1253bb39fdac5f81b08c5ab69cb06ee79345715972c1af6c5125f-usrping"
//     ],
//     "allowLists": {
//     "allowWithTags": {
//     "1": {},
//     "40000": {}
//     },
//     "allowWithoutTags": {}
//     }
//     },
//     {
//     "identifier": "raAs2Z9slHQQxwOlnniLl5aq6j+h9U8/q8sn8BIfbJADQXN5bW1Ub1B1YmxpY0JjYXN0",
//     "channelID": "raAs2Z9slHQQxwOlnniLl5aq6j+h9U8/q8sn8BIfbJAD",
//     "tags": [
//     "a164b0f40c21a86e925869e5a9d7c16886891e03f96809f68b8ade57160c7028-usrping"
//     ],
//     "allowLists": {
//     "allowWithTags": {},
//     "allowWithoutTags": {
//     "102": {},
//     "2": {}
//     }
//     }
//     },
//     {
//     "identifier": "raAs2Z9slHQQxwOlnniLl5aq6j+h9U8/q8sn8BIfbJADU3ltbWV0cmljQnJvYWRjYXN0",
//     "channelID": "raAs2Z9slHQQxwOlnniLl5aq6j+h9U8/q8sn8BIfbJAD",
//     "tags": [
//     "a164b0f40c21a86e925869e5a9d7c16886891e03f96809f68b8ade57160c7028-usrping"
//     ],
//     "allowLists": {
//     "allowWithTags": {},
//     "allowWithoutTags": {
//     "1": {},
//     "40000": {}
//     }
//     }
//     },
//     {
//     "identifier": "GNnFBbFoP+EtFdChUUEMVEOHSK2jmo+5SfTGFvu4zK4DQXN5bW1Ub1B1YmxpY0JjYXN0",
//     "channelID": "GNnFBbFoP+EtFdChUUEMVEOHSK2jmo+5SfTGFvu4zK4D",
//     "tags": [
//     "4ca23f9d385589fc530680c9e406099c6a34fd74d4002d19608ce6c971cabfa0-usrping"
//     ],
//     "allowLists": {
//     "allowWithTags": {},
//     "allowWithoutTags": {
//     "102": {},
//     "2": {}
//     }
//     }
//     },
//     {
//     "identifier": "GNnFBbFoP+EtFdChUUEMVEOHSK2jmo+5SfTGFvu4zK4DU3ltbWV0cmljQnJvYWRjYXN0",
//     "channelID": "GNnFBbFoP+EtFdChUUEMVEOHSK2jmo+5SfTGFvu4zK4D",
//     "tags": [
//     "4ca23f9d385589fc530680c9e406099c6a34fd74d4002d19608ce6c971cabfa0-usrping"
//     ],
//     "allowLists": {
//     "allowWithTags": {},
//     "allowWithoutTags": {
//     "1": {},
//     "40000": {}
//     }
//     }
//     }
//     ],
//     "changedNotificationStates": [
//     {
//     "channelID": "Z1owNo+GvizWshVW/C5IJ1izPD5oqMkCGr+PsA5If4ED",
//     "level": 20,
//     "status": 2
//     },
//     {
//     "channelID": "gZ4uFg/NaSGJVED3hH+PsezGwkZExgPeRxITlfjXZDUD",
//     "level": 40,
//     "status": 2
//     }
//     ],
//     "deletedNotificationStates": [
//     "ZGVsZXRlZAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAD"
//     ],
//     "maxState": 2
//     }
type NotificationUpdateJson struct {
	NotificationFilters       []channels.NotificationFilter `json:"notificationFilters"`
	ChangedNotificationStates []channels.NotificationState  `json:"changedNotificationStates"`
	DeletedNotificationStates []*id.ID                      `json:"deletedNotificationStates"`
	MaxState                  clientNotif.NotificationState `json:"maxState"`
}

// MessageReceivedJson is returned any time a message is received or updated.
// Update is true if the row is old and was edited.
//
//	{
//	  "uuid":32,
//	  "channelID":"Z1owNo+GvizWshVW/C5IJ1izPD5oqMkCGr+PsA5If4HZ",
//	  "update":false
//	}
type MessageReceivedJson struct {
	Uuid      int64  `json:"uuid"`
	ChannelID *id.ID `json:"channelID"`
	Update    bool   `json:"update"`
}

// UserMutedJson is returned for the MuteUser method of the impl.
//
//	{
//	  "channelID":"YSc2bDijXIVhmIsJk2OZQjU9ei2Dn6MS8tOpXlIaUpSV",
//	  "pubKey":"hClzdWkMI+LM7KDFxC/iuyIc0oiMzcBXBFgH0haZAjc=",
//	  "unmute":false
//	}
type UserMutedJson struct {
	ChannelID *id.ID            `json:"channelID"`
	PubKey    ed25519.PublicKey `json:"pubKey"`
	Unmute    bool              `json:"unmute"`
}

// MessageDeletedJson is returned any time a message is deleted.
//
//	{
//	  "messageID":"i9b7tL5sUmObxqW1LApC9H/yvnQzsRfq7yc8SCBtlK0="
//	}
type MessageDeletedJson struct {
	MessageID message.ID `json:"messageID"`
}

// AdminKeysUpdateJson describes when you get or lose keys for a specific
// channel
//
//	{
//	 "channelID":"KdkEjm+OfQuK4AyZGAqh+XPQaLfRhsO5d2NT1EIScyJX",
//	 "hasKeys":true
//	}
type AdminKeysUpdateJson struct {
	ChannelId *id.ID `json:"channelID"`
	isAdmin   bool   `json:"isAdmin"`
}

// DmTokenUpdateJson describes when the sending of dm tokens is enabled or
// disabled on a specific channel
//
//	{
//	 "channelID":"KdkEjm+OfQuK4AyZGAqh+XPQaLfRhsO5d2NT1EIScyJX",
//	 "sendToken":true
//	}
type DmTokenUpdateJson struct {
	ChannelId *id.ID `json:"channelID"`
	SendToken bool   `json:"sendToken"`
}

type ChannelUICallbacks interface {
	EventUpdate(eventType int64, jsonData []byte)
}

type ChannelUICallbacksWrapper struct {
	Cuic ChannelUICallbacks
}

func (cuicbw *ChannelUICallbacksWrapper) DmTokenUpdate(chID *id.ID, sendToken bool) {

	dmtJson := &DmTokenUpdateJson{
		ChannelId: chID,
		SendToken: sendToken,
	}
	jsonBytes, err := json.Marshal(dmtJson)
	if err != nil {
		jww.ERROR.Printf("Failed to json dm token update "+
			"event for bindings: %+v", err)
	}

	cuicbw.Cuic.EventUpdate(DmTokenUpdate, jsonBytes)
}

func (cuicbw *ChannelUICallbacksWrapper) AdminKeysUpdate(chID *id.ID, isAdmin bool) {

	akJson := &AdminKeysUpdateJson{
		ChannelId: chID,
		isAdmin:   isAdmin,
	}
	jsonBytes, err := json.Marshal(akJson)
	if err != nil {
		jww.ERROR.Printf("Failed to json admin keys update "+
			"event for bindings: %+v", err)
	}

	cuicbw.Cuic.EventUpdate(AdminKeyUpdate, jsonBytes)
}

func (cuicbw *ChannelUICallbacksWrapper) NicknameUpdate(channelId *id.ID,
	nickname string, exists bool) {

	jsonable := NickNameUpdateJson{
		ChannelId: channelId,
		Nickname:  nickname,
		Exists:    exists,
	}

	jsonBytes, err := json.Marshal(&jsonable)

	if err != nil {
		jww.ERROR.Printf("Failed to json nickname update "+
			"event for bindings: %+v", err)
	}

	cuicbw.Cuic.EventUpdate(NickNameUpdate, jsonBytes)
}

func (cuicbw *ChannelUICallbacksWrapper) NotificationUpdate(
	nfs []channels.NotificationFilter,
	changedNotificationStates []channels.NotificationState,
	deletedNotificationStates []*id.ID, maxState clientNotif.NotificationState) {

	jsonable := NotificationUpdateJson{
		NotificationFilters:       nfs,
		ChangedNotificationStates: changedNotificationStates,
		DeletedNotificationStates: deletedNotificationStates,
		MaxState:                  maxState,
	}

	jsonBytes, err := json.Marshal(&jsonable)
	if err != nil {
		jww.ERROR.Printf("Failed to json notifications update "+
			"event for bindings: %+v", err)
	}

	cuicbw.Cuic.EventUpdate(NotificationUpdate, jsonBytes)
}

func (cuicbw *ChannelUICallbacksWrapper) MessageReceived(uuid int64,
	channelID *id.ID, update bool) {

	jsonable := MessageReceivedJson{
		Uuid:      uuid,
		ChannelID: channelID,
		Update:    update,
	}

	jsonBytes, err := json.Marshal(&jsonable)
	if err != nil {
		jww.ERROR.Printf("Failed to json MessageReceived "+
			"event for bindings: %+v", err)
	}

	cuicbw.Cuic.EventUpdate(MessageReceived, jsonBytes)
}

func (cuicbw *ChannelUICallbacksWrapper) UserMuted(channelID *id.ID,
	pubKey ed25519.PublicKey, unmute bool) {
	jsonable := UserMutedJson{
		ChannelID: channelID,
		PubKey:    pubKey,
		Unmute:    unmute,
	}

	jsonBytes, err := json.Marshal(&jsonable)
	if err != nil {
		jww.ERROR.Printf("Failed to json UserMuted "+
			"event for bindings: %+v", err)
	}

	cuicbw.Cuic.EventUpdate(UserMuted, jsonBytes)
}

func (cuicbw *ChannelUICallbacksWrapper) MessageDeleted(messageID message.ID) {
	jsonable := MessageDeletedJson{MessageID: messageID}
	jsonBytes, err := json.Marshal(&jsonable)
	if err != nil {
		jww.ERROR.Printf("Failed to json MessageDeleted "+
			"event for bindings: %+v", err)
	}

	cuicbw.Cuic.EventUpdate(MessageDeleted, jsonBytes)
}
