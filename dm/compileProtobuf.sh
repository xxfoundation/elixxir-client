#!/bin/bash
################################################################################
## Copyright © 2022 xx foundation                                             ##
##                                                                            ##
## Use of this source code is governed by a license that can be found in the  ##
## LICENSE file.                                                              ##
################################################################################

# This script will compile the Protobuf file to a Go file (pb.go).
# This is meant to be called from the top level of the repo.

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )
cd $SCRIPT_DIR || return

protoc --go_out=. --go_opt=paths=source_relative ./directMessages.proto
