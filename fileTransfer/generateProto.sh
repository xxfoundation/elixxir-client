#!/bin/bash

protoc --go_out=paths=source_relative:. fileTransfer/ftMessages.proto
