package bots

import (
	"gitlab.com/elixxir/client/cmixproto"
	"gitlab.com/elixxir/client/globals"
	"gitlab.com/elixxir/client/io"
	"gitlab.com/elixxir/client/parse"
	"gitlab.com/elixxir/client/storage"
	"gitlab.com/elixxir/client/user"
	"gitlab.com/elixxir/primitives/switchboard"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
)

var session user.Session
var sessionV2 storage.Session
var topology *connect.Circuit
var comms io.Communications
var transmissionHost *connect.Host

type channelResponseListener chan string

func (l *channelResponseListener) Hear(msg switchboard.Item, isHeardElsewhere bool, i ...interface{}) {
	m := msg.(*parse.Message)
	*l <- string(m.Body)
}

var pushKeyResponseListener channelResponseListener
var getKeyResponseListener channelResponseListener
var registerResponseListener channelResponseListener
var searchResponseListener channelResponseListener
var nicknameResponseListener channelResponseListener

// Nickname request listener
type nickReqListener struct {
	MyNick string
}

// Nickname listener simply replies with message containing user's nick
func (l *nickReqListener) Hear(msg switchboard.Item, isHeardElsewhere bool, i ...interface{}) {
	m := msg.(*parse.Message)
	nick := l.MyNick
	resp := parse.Pack(&parse.TypedBody{
		MessageType: int32(cmixproto.Type_NICKNAME_RESPONSE),
		Body:        []byte(nick),
	})
	globals.Log.DEBUG.Printf("Sending nickname response to user %v", *m.Sender)
	sendCommand(m.Sender, resp)
}

var nicknameRequestListener nickReqListener

// InitBots is called internally by the Login API
func InitBots(s user.Session, s2 storage.Session, m io.Communications,
	top *connect.Circuit, host *connect.Host) {

	userData, err := s2.GetUserData()
	if err != nil {
		globals.Log.FATAL.Panicf("Could not load userdata: %+v", err)
	}

	userNick := userData.ThisUser.Username

	// FIXME: these all need to be used in non-blocking threads if we are
	// going to do it this way...
	msgBufSize := 100
	pushKeyResponseListener = make(channelResponseListener, msgBufSize)
	getKeyResponseListener = make(channelResponseListener, msgBufSize)
	registerResponseListener = make(channelResponseListener, msgBufSize)
	searchResponseListener = make(channelResponseListener, msgBufSize)
	nicknameRequestListener = nickReqListener{
		MyNick: userNick,
	}
	nicknameResponseListener = make(channelResponseListener, msgBufSize)

	session = s
	sessionV2 = s2
	topology = top
	comms = m
	transmissionHost = host

	l := m.GetSwitchboard()

	l.Register(&id.UDB, int32(cmixproto.Type_UDB_PUSH_KEY_RESPONSE),
		&pushKeyResponseListener)
	l.Register(&id.UDB, int32(cmixproto.Type_UDB_GET_KEY_RESPONSE),
		&getKeyResponseListener)
	l.Register(&id.UDB, int32(cmixproto.Type_UDB_REGISTER_RESPONSE),
		&registerResponseListener)
	l.Register(&id.UDB, int32(cmixproto.Type_UDB_SEARCH_RESPONSE),
		&searchResponseListener)
	l.Register(&id.ZeroUser,
		int32(cmixproto.Type_NICKNAME_REQUEST), &nicknameRequestListener)
	l.Register(&id.ZeroUser,
		int32(cmixproto.Type_NICKNAME_RESPONSE), &nicknameResponseListener)
}

// sendCommand sends a command to the udb. This doesn't block.
// Callers that need to wait on a response should implement waiting with a
// listener.
func sendCommand(botID *id.ID, command []byte) error {
	return comms.SendMessage(session, topology, botID,
		parse.Unencrypted, command, transmissionHost)
}

// Nickname Lookup function
func LookupNick(user *id.ID) (string, error) {
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
