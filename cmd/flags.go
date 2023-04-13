////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package cmd

// This is a comprehensive list of CLI flag name constants. Organized by
// subcommand, with root level CLI flags at the top of the list. Newly added
// flags for any existing or new subcommands should be listed and organized
// here. Pulling flags using Viper should use the constants defined here.
// todo: fill this with all existing flags, replace hardcoded references with
//
//	these constants. This makes renaming them easier, as well as having
//	a consolidated place in code for these flags.
const (
	//////////////// Root flags ///////////////////////////////////////////////

	// Send/receive flags
	verifySendFlag   = "verify-sends"
	messageFlag      = "message"
	destIdFlag       = "destid"
	sendCountFlag    = "sendCount"
	sendDelayFlag    = "sendDelay"
	splitSendsFlag   = "splitSends"
	receiveCountFlag = "receiveCount"
	waitTimeoutFlag  = "waitTimeout"
	unsafeFlag       = "unsafe"

	// Channel flags
	unsafeChannelCreationFlag = "unsafe-channel-creation"
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
	logLevelFlag = "logLevel"
	logFlag      = "log"

	// Loading/establishing xxdk.E2E
	sessionFlag       = "session"
	passwordFlag      = "password"
	ndfFlag           = "ndf"
	regCodeFlag       = "regcode"
	protoUserPathFlag = "protoUserPath"
	protoUserOutFlag  = "protoUserOut"
	forceLegacyFlag   = "force-legacy"

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
	batchMessagePickupFlag      = "batchMessagePickup"
	maxPickupBatchSizeFlag      = "maxPickupBatchSize"
	batchPickupDelayFlag        = "batchPickupDelay"
	batchPickupTimeoutFlag      = "batchPickupTimeout"

	// E2E Params
	e2eMinKeysFlag        = "e2eMinKeys"
	e2eMaxKeysFlag        = "e2eMaxKeys"
	e2eNumReKeysFlag      = "e2eNumReKeys"
	e2eRekeyThresholdFlag = "e2eRekeyThreshold"

	// Misc
	sendIdFlag                  = "sendid"
	profileCpuFlag              = "profile-cpu"
	profileMemFlag              = "profile-mem"
	userIdPrefixFlag            = "userid-prefix"
	legacyFlag                  = "legacy"
	gatewayWhitelistFlag        = "gateway-whitelist"
	disableNodeRegistrationFlag = "disableNodeRegistration"
	enableImmediateSendingFlag  = "enableImmediateSending"

	///////////////// Broadcast subcommand flags //////////////////////////////
	broadcastNameFlag        = "channelName"
	broadcastRsaPubFlag      = "rsaPub"
	broadcastSaltFlag        = "salt"
	broadcastDescriptionFlag = "description"
	broadcastChanPathFlag    = "chanPath"
	broadcastKeyPathFlag     = "keyPath"
	broadcastNewFlag         = "new"
	broadcastFlag            = "broadcast"
	broadcastSymmetricFlag   = "symmetric"
	broadcastAsymmetricFlag  = "asymmetric"

	///////////////// Channels subcommand flags ///////////////////////////////
	channelsNameFlag             = "channelName"
	channelsChanIdentityPathFlag = "channelIdentityPath"
	channelsChanPathFlag         = "channelPath"
	channelsDescriptionFlag      = "channelDescription"
	channelsJoinFlag             = "joinChannel"
	channelsKeyPathFlag          = "channelKeyPath"
	channelsLeaveFlag            = "leaveChannel"
	channelsNewFlag              = "newChannel"
	channelsSendFlag             = "sendToChannel"

	///////////////// Connection subcommand flags /////////////////////////////
	connectionFlag              = "connect"
	connectionStartServerFlag   = "startServer"
	connectionServerTimeoutFlag = "serverTimeout"
	connectionAuthenticatedFlag = "authenticated"
	connectionEphemeralFlag     = "ephemeral"

	///////////////// File Transfer subcommand flags //////////////////////////
	fileSendFlag          = "sendFile"
	filePathFlag          = "filePath"
	fileTypeFlag          = "fileType"
	filePreviewPathFlag   = "filePreviewPath"
	filePreviewStringFlag = "filePreviewString"
	fileMaxThroughputFlag = "maxThroughput"
	fileRetry             = "retry"

	///////////////// GetNdf subcommand flags //////////////////////////////
	ndfGwHostFlag   = "gwhost"
	ndfPermHostFlag = "permhost"
	ndfCertFlag     = "cert"
	ndfEnvFlag      = "env"

	///////////////// Group subcommand flags //////////////////////////////////
	groupCreateFlag         = "create"
	groupNameFlag           = "name"
	groupResendFlag         = "resend"
	groupJoinFlag           = "join"
	groupLeaveFlag          = "leave"
	groupSendMessageFlag    = "sendMessage"
	groupWaitFlag           = "wait"
	groupReceiveTimeoutFlag = "receiveTimeout"
	groupListFlag           = "list"
	groupShowFlag           = "show"

	///////////////// Single subcommand flags /////////////////////////////////
	singleSendFlag        = "send"
	singleReplyFlag       = "reply"
	singleContactFlag     = "contact"
	singleTagFlag         = "tag"
	singleMaxMessagesFlag = "maxMessages"
	singleTimeoutFlag     = "timeout"

	///////////////// User Discovery subcommand flags /////////////////////////
	udRegisterFlag       = "register"
	udRemoveFlag         = "remove"
	udAddPhoneFlag       = "addphone"
	udAddEmailFlag       = "addemail"
	udConfirmFlag        = "confirm"
	udLookupFlag         = "lookup"
	udSearchUsernameFlag = "searchusername"
	udSearchEmailFlag    = "searchemail"
	udSearchPhoneFlag    = "searchphone"
	udBatchAddFlag       = "batchadd"

	///////////////// pickup subcommand flags //////////////////////////////
	pickupGW    = "gateway"
	pickupID    = "id"
	pickupEphID = "ephid"
)
