#!/bin/bash

#///////////////////////////////////////////////////////////////////////////////
#/ Copyright © 2020 xx network SEZC                                           //
#/                                                                            //
#/ Use of this source code is governed by a license that can be found in the  //
#/ LICENSE file                                                               //
#///////////////////////////////////////////////////////////////////////////////

protoc --go_out=paths=source_relative:. channels/messages.proto
