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
	DestIdFlag       = "destid"
	SendCountFlag    = "sendCount"
	SendDelayFlag    = "sendDelay"
	SplitSendsFlag   = "splitSends"
	ReceiveCountFlag = "receiveCount"
	WaitTimeoutFlag  = "waitTimeout"
	UnsafeFlag       = "unsafe"

	// Channel flags
	UnsafeChannelCreationFlag = "unsafe-channel-creation"
	AcceptChannelFlag         = "accept-channel"
	DeleteChannelFlag         = "delete-channel"

	// Request flags
	DeleteReceiveRequestsFlag = "delete-receive-requests"
	DeleteSentRequestsFlag    = "delete-sent-requests"
	DeleteAllRequestsFlag     = "delete-all-requests"
	DeleteRequestFlag         = "delete-request"
	SendAuthRequestFlag       = "send-auth-request"
	AuthTimeoutFlag           = "auth-timeout"

	// Contact file flags
	WriteContactFlag = "writeContact"
	DestFileFlag     = "destfile"

	// Log flags
	LogLevelFlag = "logLevel"
	LogFlag      = "log"

	// Loading/establishing xxdk.E2E
	SessionFlag       = "session"
	PasswordFlag      = "password"
	NdfFlag           = "ndf"
	RegCodeFlag       = "regcode"
	ProtoUserPathFlag = "protoUserPath"
	ProtoUserOutFlag  = "protoUserOut"
	ForceLegacyFlag   = "force-legacy"

	// Backup flags
	BackupOutFlag     = "backupOut"
	BackupJsonOutFlag = "backupJsonOut"
	BackupInFlag      = "backupIn"
	BackupPassFlag    = "backupPass"
	BackupIdListFlag  = "backupIdList"

	// Network following/logging flags
	VerboseRoundTrackingFlag    = "verboseRoundTracking"
	ForceHistoricalRoundsFlag   = "forceHistoricalRounds"
	SlowPollingFlag             = "slowPolling"
	ForceMessagePickupRetryFlag = "forceMessagePickupRetry"

	// E2E Params
	E2eMinKeysFlag        = "e2eMinKeys"
	E2eMaxKeysFlag        = "e2eMaxKeys"
	E2eNumReKeysFlag      = "e2eNumReKeys"
	E2eRekeyThresholdFlag = "e2eRekeyThreshold"

	// Misc
	SendIdFlag       = "sendid"
	ProfileCpuFlag   = "profile-cpu"
	userIdPrefixFlag = "userid-prefix"

	///////////////// GetNdf subcommand flags //////////////////////////////
	NdfGwHostFlag   = "gwhost"
	NdfPermHostFlag = "permhost"
	NdfCertFlag     = "cert"
	NdfEnvFlag      = "env"
)
