////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package bindings

import (
	"encoding/binary"
	"errors"
	"gitlab.com/privategrity/client/api"
	"gitlab.com/privategrity/client/globals"
	"gitlab.com/privategrity/crypto/cyclic"
	"github.com/xeipuuv/gojsonschema"
	"strconv"
	"github.com/spf13/jwalterweatherman"
)

// Copy of the storage interface.
// It is identical to the interface used in Globals,
// and a results the types can be passed freely between the two
type Storage interface {
	SetLocation(string) (Storage, error)
	GetLocation() string
	Save([]byte) (Storage, error)
	Load() []byte
}

//Message used for binding
type Message struct {
	Sender    []byte
	Payload   string
	Recipient []byte
}

// Initializes the client by registering a storage mechanism.
// For the mobile interface, one must be provided
func InitClient(s Storage, loc string) error {

	if s == nil {
		return errors.New("could not init client")
	}

	storeState := api.InitClient(s.(globals.Storage), loc)

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
		Sender:    cyclic.NewIntFromBytes(m.Sender).Uint64(),
		Payload:   m.Payload,
		Recipient: cyclic.NewIntFromBytes(m.Recipient).Uint64(),
	}

	return api.Send(apiMsg)
}

func TryReceive() (Message, error) {
	m, err := api.TryReceive()

	var msg Message

	if err != nil {
		msg = Message{
			Sender:    cyclic.NewIntFromUInt(m.Sender).Bytes(),
			Payload:   m.Payload,
			Recipient: cyclic.NewIntFromUInt(m.Recipient).Bytes(),
		}
	}

	return msg, err
}

func Logout() error {
	return api.Logout()
}

// Byte order for our APIs are conventionally going to be little-endian
func SetNick(UID []byte, nick string) error {
	return api.SetNick(binary.LittleEndian.Uint64(UID), nick)
}

func UpdateContactList() {
	api.UpdateContactList()
}

// We use this schema to validate the JSON we've generated at runtime,
// and users of the bindings can use it as a description of the data they'll get
// when they get the contact list.
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

func buildContactListJSON(ids []uint64, nicks []string) ([]byte) {
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


func validateContactListJSON(json []byte) error {
	if contactListSchemaCreationError != nil {
		jwalterweatherman.ERROR.Printf(
			"Couldn't instantiate JSON schema: %v", contactListSchemaCreationError.Error())
		return contactListSchemaCreationError
	}

	jsonLoader := gojsonschema.NewBytesLoader(json)
	valid, err := contactListSchema.Validate(jsonLoader)

	if err != nil {
		annotatedError := errors.New("Failed to validate JSON: " + err.Error())
		jwalterweatherman.ERROR.Println(annotatedError.Error())
		return annotatedError
	}
	if !valid.Valid() {
		for _, validationError := range (valid.Errors()) {
			annotatedError := errors.New(
				"The produced JSON wasn't valid" + validationError.String())
			jwalterweatherman.ERROR.Println(annotatedError.Error())
			return annotatedError
		}
	}

	return nil
}

func GetContactListJSON() ([]byte, error) {
	ids, nicks := api.GetContactList()
	result := buildContactListJSON(ids, nicks)
	err := validateContactListJSON(result)
	return result, err
}
