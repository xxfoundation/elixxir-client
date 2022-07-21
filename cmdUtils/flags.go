///////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

package cmd

import (
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
)

// This is a list of CLI flag name constants shared between root and subcommands.
// Newly added flags for any flag which should be accessible by more than one command
// should be listed here. Flags designed for a specific subcommand should go in its
// respective cmd directory. Pulling flags using Viper should use the constants defined here.
const (

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
	EphemeralFlag     = "ephemeral"

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
	UserIdPrefixFlag = "userid-prefix"

	///////////////// GetNdf subcommand flags //////////////////////////////

	NdfGwHostFlag   = "gwhost"
	NdfPermHostFlag = "permhost"
	NdfCertFlag     = "cert"
	NdfEnvFlag      = "env"
)

// BindFlagHelper binds the key to a pflag.Flag used by Cobra and prints an
// error if one occurs.
func BindFlagHelper(key string, command *cobra.Command) {
	err := viper.BindPFlag(key, command.Flags().Lookup(key))
	if err != nil {
		jww.ERROR.Printf("viper.BindPFlag failed for %q: %+v", key, err)
	}
}

// BindPersistentFlagHelper binds the key to a Persistent pflag.Flag used by Cobra and prints an
// error if one occurs.
func BindPersistentFlagHelper(key string, command *cobra.Command) {
	err := viper.BindPFlag(key, command.PersistentFlags().Lookup(key))
	if err != nil {
		jww.ERROR.Printf("viper.BindPFlag failed for %q: %+v", key, err)
	}
}
