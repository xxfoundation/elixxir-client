///////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package cmd

// Flag name constants
// todo: fill this with all existing flags, replace hardcoded references with
//  these constants
const (
	//////////////// Root flags ///////////////////////////////////////////////

	// Send flags
	verifySendFlag = "verify-sends"
	destFileFlag   = "destfile"
	messageFlag    = "message"

	// Log flags
	logLevelFlag = "logLevel"
	logFlag      = "log"
	sessionFlag  = "session"

	///////////////// Broadcast subcommand flags //////////////////////////////
	// todo: populate

	///////////////// Connection subcommand flags /////////////////////////////
	connectionFlag    = "connect"
	startServerFlag   = "startServer"
	serverTimeoutFlag = "serverTimeoutFlag"
	disconnectFlag    = "disconnect"
	authenticatedFlag = "authenticated"

	///////////////// File Transfer subcommand flags //////////////////////////
	// todo: populate

	///////////////// Group subcommand flags //////////////////////////////////
	// todo: populate

	///////////////// Single subcommand flags /////////////////////////////////
	// todo: populate

	///////////////// User Discovery subcommand flags /////////////////////////
	// todo: populate

)
