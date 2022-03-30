#!/bin/bash

protoc --go_out=. -I../ -I$PWD --go_opt=paths=source_relative xchange.proto
