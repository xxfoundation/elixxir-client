#!/bin/bash

protoc --go_out=. -I$GOPATH/src/gitlab.com/privategrity/ channelbot/messageTypes.proto user-discovery-bot/messageTypes.proto
