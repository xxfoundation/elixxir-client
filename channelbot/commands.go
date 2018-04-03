package channelbot

import (
	"strings"
	"errors"
	"fmt"
	"strconv"
)

// RunAdd adds the passed user to the channel bot
// It takes the user's ID in base 10
func RunAdd(params []string, senderId uint64) error {
	if len(params) != 1 {
		return errors.New("\add takes one parameter: the ID of the user who" +
			"'ll be added to the channel.")
	} else {
		id, err := strconv.ParseUint(params[0], 10, 64)
		if err != nil {
			return errors.New("\add couldn't parse the user ID: " + err.Error())
		}
		return AddUser(id, senderId)
	}
}

// RunRemove removes the passed user from the channel bot
// It takes the user's ID in base 10
func RunRemove(params []string, senderId uint64) error {
	if len(params) != 1 {
		return errors.New("\remove takes one parameter: the ID of the user" +
			" who'll be removed from the channel.")
	} else {
		id, err := strconv.ParseUint(params[0], 10, 64)
		if err != nil {
			return errors.New("\remove couldn't parse the user ID: " + err.
				Error())
		}
		return RemoveUser(id, senderId)
	}
}

func ParseCommand(command string, senderId uint64) error {
	if command == "" {
		return errors.New("ParseCommand: Can't parse empty string")
	}
	if command[0] == '/' {
		// this is, indeed, a command, so we should parse it
		tokens := strings.Fields(command)
		switch tokens[0] {
		case "/add":
			return RunAdd(tokens[1:], senderId)
		case "/remove":
			return RunRemove(tokens[1:], senderId)
		default:
			return errors.New(fmt.Sprintf(
				"ParseCommand: Unrecognized command %v", tokens[0]))
		}
	} else {
		return errors.New("ParseCommand: this isn't a command")
	}
}
