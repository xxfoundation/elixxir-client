#!/bin/bash

protoc --go_out=paths=source_relative:. restlike/restLikeMessages.proto
