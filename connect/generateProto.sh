#!/bin/bash

# This script will generate the protobuf Golang file (pb.go) out of the protobuf file (.proto).
# This is meant to be called from the top level of the repo.

protoc --go_out=paths=source_relative:. connections/authenticated/authenticated.proto
