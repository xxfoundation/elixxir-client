///////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package cmd

// This is a comprehensive list of CLI flag name constants. Organized by
// subcommand, with root level CLI flags at the top of the list. Newly added
// flags for any existing or new subcommands should be listed and organized
// here. Pulling flags using Viper should use the constants defined here.
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

	// Misc
	sessionFlag = "session"

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
