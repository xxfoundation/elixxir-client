package bots

import (
	"gitlab.com/elixxir/client/cmixproto"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/io"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/client/user"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/primitives/circuit"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/switchboard"
)

var session user.Session
var topology *circuit.Circuit
var comms io.Communications
var transmissionHost *connect.Host

// UdbID is the ID of the user discovery bot, which is always 3
var UdbID *id.User

type channelResponseListener chan string

func (l *channelResponseListener) Hear(msg switchboard.Item, isHeardElsewhere bool) {
	m := msg.(*parse.Message)
	*l <- string(m.Body)
}

var pushKeyResponseListener channelResponseListener
var getKeyResponseListener channelResponseListener
var registerResponseListener channelResponseListener
var searchResponseListener channelResponseListener
var nicknameResponseListener channelResponseListener

// Nickname request listener
type nickReqListener struct{}

// Nickname listener simply replies with message containing user's nick
func (l *nickReqListener) Hear(msg switchboard.Item, isHeardElsewhere bool) {
	m := msg.(*parse.Message)
	nick := session.GetCurrentUser().Nick
	resp := parse.Pack(&parse.TypedBody{
		MessageType: int32(cmixproto.Type_NICKNAME_RESPONSE),
		Body:        []byte(nick),
	})
	globals.Log.DEBUG.Printf("Sending nickname response to user %v", *m.Sender)
	sendCommand(m.Sender, resp)
}

var nicknameRequestListener nickReqListener

// InitBots is called internally by the Login API
func InitBots(s user.Session, m io.Communications, top *circuit.Circuit, udbID *id.User, host *connect.Host) {
	UdbID = udbID

	// FIXME: these all need to be used in non-blocking threads if we are
	// going to do it this way...
	msgBufSize := 100
	pushKeyResponseListener = make(channelResponseListener, msgBufSize)
	getKeyResponseListener = make(channelResponseListener, msgBufSize)
	registerResponseListener = make(channelResponseListener, msgBufSize)
	searchResponseListener = make(channelResponseListener, msgBufSize)
	nicknameRequestListener = nickReqListener{}
	nicknameResponseListener = make(channelResponseListener, msgBufSize)

	session = s
	topology = top
	comms = m
	transmissionHost = host

	l := session.GetSwitchboard()

	l.Register(UdbID, int32(cmixproto.Type_UDB_PUSH_KEY_RESPONSE),
		&pushKeyResponseListener)
	l.Register(UdbID, int32(cmixproto.Type_UDB_GET_KEY_RESPONSE),
		&getKeyResponseListener)
	l.Register(UdbID, int32(cmixproto.Type_UDB_REGISTER_RESPONSE),
		&registerResponseListener)
	l.Register(UdbID, int32(cmixproto.Type_UDB_SEARCH_RESPONSE),
		&searchResponseListener)
	l.Register(id.ZeroID,
		int32(cmixproto.Type_NICKNAME_REQUEST), &nicknameRequestListener)
	l.Register(id.ZeroID,
		int32(cmixproto.Type_NICKNAME_RESPONSE), &nicknameResponseListener)
}

// sendCommand sends a command to the udb. This doesn't block.
// Callers that need to wait on a response should implement waiting with a
// listener.
func sendCommand(botID *id.User, command []byte) error {
	return comms.SendMessage(session, topology, botID,
		parse.Unencrypted, command, transmissionHost)
}

// Nickname Lookup function
func LookupNick(user *id.User) (string, error) {
	globals.Log.DEBUG.Printf("Sending nickname request to user %v", *user)
	msg := parse.Pack(&parse.TypedBody{
		MessageType: int32(cmixproto.Type_NICKNAME_REQUEST),
		Body:        []byte{},
	})

	err := sendCommand(user, msg)
	if err != nil {
		return "", err
	}

	nickResponse := <-nicknameResponseListener
	return nickResponse, nil
}
