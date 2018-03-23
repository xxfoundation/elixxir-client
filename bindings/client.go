////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"errors"
	"github.com/spf13/jwalterweatherman"
	"github.com/xeipuuv/gojsonschema"
	"gitlab.com/privategrity/client/api"
	"gitlab.com/privategrity/client/globals"
	"gitlab.com/privategrity/crypto/cyclic"
	"strconv"
	"gitlab.com/privategrity/crypto/format"
)

// Copy of the storage interface.
// It is identical to the interface used in Globals,
// and a results the types can be passed freely between the two
type Storage interface {
	// Give a Location for storage.  Does not need to be implemented if unused.
	SetLocation(string) error
	// Returns the Location for storage.
	// Does not need to be implemented if unused.
	GetLocation() string
	// Stores the passed byte slice
	Save([]byte) error
	// Returns the stored byte slice
	Load() []byte
}

//Message used for binding
type Message interface {
	// Returns the message's sender ID
	// (uint64) BigEndian serialized into a byte slice
	GetSender() []byte
	// Returns the message payload
	GetPayload() string
	// Returns the message's recipient ID
	// (uint64) BigEndian serialized into a byte slice
	GetRecipient() []byte
}

// An object implementing this interface can be called back when the client
// gets a message
type Receiver interface {
	Receive(message Message)
}

// Initializes the client by registering a storage mechanism and a reception
// callback.
// For the mobile interface, one must be provided
// The loc can be empty, it is only necessary if the passed storage interface
// requires it to be passed via "SetLocation"
//
// Parameters: storage implements Storage.
// Implement this interface to store the user session data locally.
// You must give us something for this parameter.
//
// loc is a string. If you're using DefaultStorage for your storage,
// this would be the filename of the file that you're storing the user
// session in.
//
// receiver implements Receiver.
// This parameter is optional. If this parameter is null,
// you can receive messages by polling the API with TryReceive.
// If you pass a non-null object implementing Receiver in this
// parameter, we will call that Receiver when the client gets a message.
func InitClient(storage Storage, loc string, receiver Receiver) error {
	r := func (messageInterface format.MessageInterface) {
		receiver.Receive(messageInterface.(Message))
	}

	if s == nil {
		return errors.New("could not init client: Storage was nil")
	}

	err := api.InitClient(storage.(globals.Storage), loc, r)

	return err
}

// Registers user and returns the User ID.  Returns null if registration fails.
// registrationCode is a one time use string.
// nick is a nickname which must be 32 characters or less.
// nodeAddr is the ip address and port of the last node in the form: 192.168.1.1:50000
// numNodes is the number of nodes in the system
// Valid codes:
// 1
// “David”
// 2HOAAFKIVKEJ0
// 2
// “Jim”
// EPJHMGE1KHTVS
// 3
// “Ben”
// 8L7U3HHEOC04T
// 4
// “Rick”
// 4DU574DN9R292
// 5
// “Spencer”
// BE50NHQPQJTJJ
// 6
// “Jake”
// 1JB2L6A6L76KU
// 7
// “Mario”
// DEFJS3NIG55P5
// 8
// “Will”
// F2MIJJ1S8DLV6
// 9
// “Allan”
// 3GENI79B65V2A
// 10
// “Jono”
// JHJ6L9BACDVC
func Register(registrationCode string, nick string, nodeAddr string,
	numNodes int) ([]byte, error) {

	if numNodes < 1 {
		return nil, errors.New("invalid number of nodes")
	}

	hashUID := cyclic.NewIntFromString(registrationCode, 32).Uint64()

	UID, err := api.Register(hashUID, nick, nodeAddr, uint(numNodes))

	if err != nil {
		return nil, err
	}

	return cyclic.NewIntFromUInt(UID).Bytes(), nil
}

// Logs in the user based on User ID and returns the nickname of that user.
// Returns an empty string and an error
// UID is a uint64 BigEndian serialized into a byte slice
func Login(UID []byte) (string, error) {
	userID := cyclic.NewIntFromBytes(UID).Uint64()
	nick, err := api.Login(userID)
	return nick, err
}

//Sends a message structured via the message interface
func Send(m Message) error {
	return api.Send(m)
}

// Attempts to retrieve a message from the queue.
// Returns a nil message if none are available.
func TryReceive() (Message, error) {
	return api.TryReceive()
}

// Logs the user out, saving the state for the system and clearing all data
// from RAM
func Logout() error {
	return api.Logout()
}

// Byte order for our APIs are conventionally going to be little-endian
/* Set this user's nick on the server */
func SetNick(UID []byte, nick string) error {
	userID := cyclic.NewIntFromBytes(UID).Uint64()
	return api.SetNick(userID, nick)
}

/* Get an updated list of all users that the server knows about and update the
 * user structure to include all of them */
func UpdateContactList() error {
	return api.UpdateContactList()
}

/* We use this schema to validate the JSON we've generated at runtime,
 * and users of the bindings can use it as a description of the data they'll get
 * when they get the contact list. */
var ContactListJsonSchema = `{
	"type": "array",
	"items": {
		"type": "object",
		"properties": {
			"UserID": { "type": "number" },
			"Nick": { "type": "string" }
		}
	}
}`

var contactListSchema, contactListSchemaCreationError = gojsonschema.NewSchema(
	gojsonschema.NewStringLoader(ContactListJsonSchema))

/* Represent slices of UserID and Nick as JSON. ContactListJsonSchema is the
 * JSON schema that shows how the resulting data are structured. */
func buildContactListJSON(ids []uint64, nicks []string) []byte {
	var result []byte
	result = append(result, '[')
	for i := 0; i < len(ids) && i < len(nicks); i++ {
		result = append(result, `{"UserID":`...)
		result = append(result, strconv.FormatUint(ids[i], 10)...)
		result = append(result, `,"Nick":"`...)
		result = append(result, nicks[i]...)
		result = append(result, `"},`...)
	}
	// replace the last comma with a bracket, ending the list
	result[len(result)-1] = ']'

	return result
}

/* Make sure that a JSON file conforms to the schema for contact list information */
func validateContactListJSON(json []byte) error {
	// Ensure that the schema was created correctly
	if contactListSchemaCreationError != nil {
		jwalterweatherman.ERROR.Printf(
			"Couldn't instantiate JSON schema: %v", contactListSchemaCreationError.Error())
		return contactListSchemaCreationError
	}

	jsonLoader := gojsonschema.NewBytesLoader(json)
	valid, err := contactListSchema.Validate(jsonLoader)

	// Ensure that the schema could validate the JSON
	if err != nil {
		annotatedError := errors.New("Failed to validate JSON: " + err.Error())
		jwalterweatherman.ERROR.Println(annotatedError.Error())
		return annotatedError
	}
	// Ensure that the JSON matches the schema
	if !valid.Valid() {
		for _, validationError := range valid.Errors() {
			annotatedError := errors.New(
				"The produced JSON wasn't valid" + validationError.String())
			jwalterweatherman.ERROR.Println(annotatedError.Error())
			return annotatedError
		}
	}

	// No errors occurred in any of the steps, so this JSON is good.
	return nil
}

/* Gets a list of user IDs and nicks and returns them as a JSON object because
 * Gomobile has dumb limitations.
 *
 * ContactListJSONSchema is the JSON schema that shows how the resulting data
 * are structured. You'll get an array, and each element of the array has a
 * UserID which is a number, and a Nick which is a string. */
func GetContactListJSON() ([]byte, error) {
	/*updateError := api.UpdateContactList()
	if updateError != nil {
		updateError = errors.New("Update contact list failed: "+ updateError.
			Error())
	}*/
	var updateError error
	ids, nicks := api.GetContactList()
	result := buildContactListJSON(ids, nicks)
	validateError := validateContactListJSON(result)
	if validateError != nil {
		validateError = errors.New("Validate contact list failed: "+
			validateError.Error())
	}
	if updateError != nil && validateError != nil {
		return result, errors.New(updateError.Error() + validateError.Error())
	}
	if updateError != nil && validateError == nil {
		return result, updateError
	}
	return result, validateError
}

//Disables Ratcheting, only for debugging
func DisableRatchet() {
	api.DisableRatchet()
}
