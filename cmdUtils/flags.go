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
//  these constants. This makes renaming them easier, as well as having
//  a consolidated place in code for these flags.
const (
	//////////////// Root flags ///////////////////////////////////////////////

	/// Send/receive flags

	VerifySendFlag   = "verify-sends"
	MessageFlag      = "message"
	destIdFlag       = "destid"
	sendCountFlag    = "sendCount"
	SendDelayFlag    = "sendDelay"
	splitSendsFlag   = "splitSends"
	ReceiveCountFlag = "receiveCount"
	WaitTimeoutFlag  = "waitTimeout"
	unsafeFlag       = "unsafe"

	// Channel flags
	UnsafeChannelCreationFlag = "unsafe-channel-creation"
	acceptChannelFlag         = "accept-channel"
	deleteChannelFlag         = "delete-channel"

	// Request flags
	deleteReceiveRequestsFlag = "delete-receive-requests"
	deleteSentRequestsFlag    = "delete-sent-requests"
	deleteAllRequestsFlag     = "delete-all-requests"
	deleteRequestFlag         = "delete-request"
	sendAuthRequestFlag       = "send-auth-request"
	authTimeoutFlag           = "auth-timeout"

	// Contact file flags
	writeContactFlag = "writeContact"
	destFileFlag     = "destfile"

	// Log flags
	LogLevelFlag = "logLevel"
	LogFlag      = "log"

	// Loading/establishing xxdk.E2E
	SessionFlag       = "session"
	PasswordFlag      = "password"
	NdfFlag           = "ndf"
	RegCodeFlag       = "regcode"
	protoUserPathFlag = "protoUserPath"
	protoUserOutFlag  = "protoUserOut"
	ForceLegacyFlag   = "force-legacy"

	// Backup flags
	backupOutFlag     = "backupOut"
	backupJsonOutFlag = "backupJsonOut"
	backupInFlag      = "backupIn"
	backupPassFlag    = "backupPass"
	backupIdListFlag  = "backupIdList"

	// Network following/logging flags
	verboseRoundTrackingFlag    = "verboseRoundTracking"
	forceHistoricalRoundsFlag   = "forceHistoricalRounds"
	slowPollingFlag             = "slowPolling"
	forceMessagePickupRetryFlag = "forceMessagePickupRetry"

	// E2E Params
	e2eMinKeysFlag        = "e2eMinKeys"
	e2eMaxKeysFlag        = "e2eMaxKeys"
	e2eNumReKeysFlag      = "e2eNumReKeys"
	e2eRekeyThresholdFlag = "e2eRekeyThreshold"

	// Misc
	sendIdFlag       = "sendid"
	profileCpuFlag   = "profile-cpu"
	userIdPrefixFlag = "userid-prefix"
)
