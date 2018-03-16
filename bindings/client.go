////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"encoding/binary"
	"errors"
	"github.com/spf13/jwalterweatherman"
	"github.com/xeipuuv/gojsonschema"
	"gitlab.com/privategrity/client/api"
	"gitlab.com/privategrity/client/globals"
	"gitlab.com/privategrity/crypto/cyclic"
	"strconv"
)

// Copy of the storage interface.
// It is identical to the interface used in Globals,
// and a results the types can be passed freely between the two
type Storage interface {
	SetLocation(string) (*Storage, error)
	GetLocation() string
	Save([]byte) (*Storage, error)
	Load() []byte
}

//Message used for binding
type Message interface {
	GetSender() []byte
	GetPayload() string
	GetRecipient() []byte
}

// Initializes the client by registering a storage mechanism.
// For the mobile interface, one must be provided
func InitClient(s *Storage, loc string) error {

	if s == nil {
		return errors.New("could not init client")
	}

	storeState := api.InitClient((*s).(globals.Storage), loc)

	return storeState
}

//Registers user and returns the User ID.  Returns nil if registration fails.
func Register(HUID []byte, nick string, nodeAddr string,
	numNodes int) ([]byte, error) {

	if len(HUID) > 8 {
		return nil, errors.New("HUID is to long")
	}

	if numNodes < 1 {
		return nil, errors.New("invalid number of nodes")
	}

	HashUID := cyclic.NewIntFromBytes(HUID).Uint64()

	UID, err := api.Register(HashUID, nick, nodeAddr, uint(numNodes))

	if err != nil {
		return nil, err
	}

	return cyclic.NewIntFromUInt(UID).Bytes(), nil
}

// Logs in the user based on User ID and returns the nickname of that user.
// Returns an empty string if the login is unsuccessful
func Login(UID []byte) (string, error) {
	nick, err := api.Login(cyclic.NewIntFromBytes(UID).Uint64())
	return nick, err
}

func Send(m Message) error {
	apiMsg := api.APIMessage{
		Sender:    binary.LittleEndian.Uint64(m.GetSender()),
		Payload:   m.GetPayload(),
		Recipient: binary.LittleEndian.Uint64(m.GetRecipient()),
	}

	return api.Send(apiMsg)
}

func TryReceive() (Message, error) {
	message, err := api.TryReceive()
	return &message, err
}

func Logout() error {
	return api.Logout()
}

// Byte order for our APIs are conventionally going to be little-endian
/* Set this user's nick on the server */
func SetNick(UID []byte, nick string) error {
	return api.SetNick(binary.LittleEndian.Uint64(UID), nick)
}

/* Get an updated list of all users that the server knows about and update the
 * user structure to include all of them */
func UpdateContactList() {
	api.UpdateContactList()
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
	ids, nicks := api.GetContactList()
	result := buildContactListJSON(ids, nicks)
	err := validateContactListJSON(result)
	return result, err
}
