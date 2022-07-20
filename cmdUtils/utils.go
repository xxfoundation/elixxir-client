package cmd

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/e2e"
	"gitlab.com/elixxir/client/xxdk"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/primitives/excludedRounds"
	"gitlab.com/xx_network/primitives/id"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

// Key used for storing xxdk.ReceptionIdentity objects
const IdentityStorageKey = "identityStorageKey"

// BindFlagHelper binds the key to a pflag.Flag used by Cobra and prints an
// error if one occurs.
func BindFlagHelper(key string, command *cobra.Command) {
	err := viper.BindPFlag(key, command.Flags().Lookup(key))
	if err != nil {
		jww.ERROR.Printf("viper.BindPFlag failed for %q: %+v", key, err)
	}
}

func VerifySendSuccess(client *xxdk.E2e, paramsE2E e2e.Params,
	roundIDs []id.Round, partnerId *id.ID, payload []byte) bool {
	retryChan := make(chan struct{})
	done := make(chan struct{}, 1)

	// Construct the callback function which
	// verifies successful message send or retries
	f := func(allRoundsSucceeded, timedOut bool,
		rounds map[id.Round]cmix.RoundResult) {
		printRoundResults(
			rounds, roundIDs, payload, partnerId)
		if !allRoundsSucceeded {
			retryChan <- struct{}{}
		} else {
			done <- struct{}{}
		}
	}

	// Monitor rounds for results
	err := client.GetCmix().GetRoundResults(
		paramsE2E.CMIXParams.Timeout, f, roundIDs...)
	if err != nil {
		jww.DEBUG.Printf("Could not verify messages were sent " +
			"successfully, resending messages...")
		return false
	}

	select {
	case <-retryChan:
		// On a retry, go to the top of the loop
		jww.DEBUG.Printf("Messages were not sent successfully," +
			" resending messages...")
		return false
	case <-done:
		// Close channels on verification success
		close(done)
		close(retryChan)
		return true
	}
}

//////////////////////////////////////////////////////////////////////////////////////
// Logging and parameters
//////////////////////////////////////////////////////////////////////////////////////

func InitParams() (xxdk.CMIXParams, xxdk.E2EParams) {
	e2eParams := xxdk.GetDefaultE2EParams()
	e2eParams.Session.MinKeys = uint16(viper.GetUint(e2eMinKeysFlag))
	e2eParams.Session.MaxKeys = uint16(viper.GetUint(e2eMaxKeysFlag))
	e2eParams.Session.NumRekeys = uint16(viper.GetUint(e2eNumReKeysFlag))
	e2eParams.Session.RekeyThreshold = viper.GetFloat64(e2eRekeyThresholdFlag)

	if viper.GetBool("splitSends") {
		e2eParams.Base.ExcludedRounds = excludedRounds.NewSet()
	}

	cmixParams := xxdk.GetDefaultCMixParams()
	cmixParams.Network.Pickup.ForceHistoricalRounds = viper.GetBool(
		forceHistoricalRoundsFlag)
	cmixParams.Network.FastPolling = !viper.GetBool(slowPollingFlag)
	cmixParams.Network.Pickup.ForceMessagePickupRetry = viper.GetBool(
		forceMessagePickupRetryFlag)
	if cmixParams.Network.Pickup.ForceMessagePickupRetry {
		period := 3 * time.Second
		jww.INFO.Printf("Setting Uncheck Round Period to %v", period)
		cmixParams.Network.Pickup.UncheckRoundPeriod = period
	}
	cmixParams.Network.VerboseRoundTracking = viper.GetBool(
		verboseRoundTrackingFlag)
	return cmixParams, e2eParams
}

func InitLog(threshold uint, logPath string) {
	if logPath != "-" && logPath != "" {
		// Disable stdout output
		jww.SetStdoutOutput(ioutil.Discard)
		// Use log file
		logOutput, err := os.OpenFile(logPath,
			os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic(err.Error())
		}
		jww.SetLogOutput(logOutput)
	}

	if threshold > 1 {
		jww.INFO.Printf("log level set to: TRACE")
		jww.SetStdoutThreshold(jww.LevelTrace)
		jww.SetLogThreshold(jww.LevelTrace)
		jww.SetFlags(log.LstdFlags | log.Lmicroseconds)
	} else if threshold == 1 {
		jww.INFO.Printf("log level set to: DEBUG")
		jww.SetStdoutThreshold(jww.LevelDebug)
		jww.SetLogThreshold(jww.LevelDebug)
		jww.SetFlags(log.LstdFlags | log.Lmicroseconds)
	} else {
		jww.INFO.Printf("log level set to: INFO")
		jww.SetStdoutThreshold(jww.LevelInfo)
		jww.SetLogThreshold(jww.LevelInfo)
	}

	if viper.GetBool(verboseRoundTrackingFlag) {
		initRoundLog(logPath)
	}

	jww.INFO.Printf(version())
}

func version() string {
	out := fmt.Sprintf("Elixxir Cmix v%s -- %s\n\n", xxdk.SEMVER,
		xxdk.GITVERSION)
	out += fmt.Sprintf("Dependencies:\n\n%s\n", xxdk.DEPENDENCIES)
	return out
}

// initRoundLog creates the log output for round tracking. In debug mode,
// the client will keep track of all rounds it evaluates if it has
// messages in, and then will dump them to this log on client exit
func initRoundLog(logPath string) *jww.Notepad {
	parts := strings.Split(logPath, ".")
	path := parts[0] + "-rounds." + parts[1]
	logOutput, err := os.OpenFile(path,
		os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		jww.FATAL.Panicf(err.Error())
	}
	return jww.NewNotepad(jww.LevelInfo, jww.LevelInfo,
		ioutil.Discard, logOutput, "", log.Ldate|log.Ltime)
}

////////////////////////////////////////////////////////////////////////////////////////
// File IO Utils
///////////////////////////////////////////////////////////////////////////////////////

// GetContactFromFile loads the contact from the given file path.
func GetContactFromFile(path string) contact.Contact {
	data, err := ioutil.ReadFile(path)
	jww.INFO.Printf("Read in contact file of size %d bytes", len(data))
	if err != nil {
		jww.FATAL.Panicf("Failed to read contact file: %+v", err)
	}

	c, err := contact.Unmarshal(data)
	if err != nil {
		jww.FATAL.Panicf("Failed to unmarshal contact: %+v", err)
	}

	return c
}

func WriteContact(c contact.Contact) {
	outfilePath := viper.GetString(writeContactFlag)
	if outfilePath == "" {
		return
	}
	jww.INFO.Printf("PubKey WRITE: %s", c.DhPubKey.Text(10))
	err := ioutil.WriteFile(outfilePath, c.Marshal(), 0644)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
}

// Helper function which prints the round results
func printRoundResults(rounds map[id.Round]cmix.RoundResult, roundIDs []id.Round, payload []byte, recipient *id.ID) {

	// Done as string slices for easy and human-readable printing
	successfulRounds := make([]string, 0)
	failedRounds := make([]string, 0)
	timedOutRounds := make([]string, 0)

	for _, r := range roundIDs {
		// Group all round reports into a category based on their
		// result (successful, failed, or timed out)
		if result, exists := rounds[r]; exists {
			if result.Status == cmix.Succeeded {
				successfulRounds = append(successfulRounds, strconv.Itoa(int(r)))
			} else if result.Status == cmix.Failed {
				failedRounds = append(failedRounds, strconv.Itoa(int(r)))
			} else {
				timedOutRounds = append(timedOutRounds, strconv.Itoa(int(r)))
			}
		}
	}

	jww.INFO.Printf("Result of sending message \"%s\" to \"%v\":",
		payload, recipient)

	// Print out all rounds results, if they are populated
	if len(successfulRounds) > 0 {
		jww.INFO.Printf("\tRound(s) %v successful", strings.Join(successfulRounds, ","))
	}
	if len(failedRounds) > 0 {
		jww.ERROR.Printf("\tRound(s) %v failed", strings.Join(failedRounds, ","))
	}
	if len(timedOutRounds) > 0 {
		jww.ERROR.Printf("\tRound(s) %v timed out (no network resolution could be found)",
			strings.Join(timedOutRounds, ","))
	}
}

///////////////////////////////////////////////////////////////////////////////////////////
// Channel Helper Functions
//////////////////////////////////////////////////////////////////////////////////////////

func acceptChannelVerified(messenger *xxdk.E2e, recipientID *id.ID,
	params xxdk.E2EParams) {
	for {
		rid := acceptChannel(messenger, recipientID)
		VerifySendSuccess(messenger, params.Base, []id.Round{rid}, recipientID, nil)
	}
}

func acceptChannel(messenger *xxdk.E2e, recipientID *id.ID) id.Round {
	recipientContact, err := messenger.GetAuth().GetReceivedRequest(
		recipientID)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	rid, err := messenger.GetAuth().Confirm(
		recipientContact)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}

	return rid
}

/////////////////////////////////////////////////////////////////////////////////////
// Password logic
////////////////////////////////////////////////////////////////////////////////////

func ParsePassword(pwStr string) []byte {
	if strings.HasPrefix(pwStr, "0x") {
		return getPWFromHexString(pwStr[2:])
	} else if strings.HasPrefix(pwStr, "b64:") {
		return getPWFromb64String(pwStr[4:])
	} else {
		return []byte(pwStr)
	}
}

func getPWFromb64String(pwStr string) []byte {
	pwBytes, err := base64.StdEncoding.DecodeString(pwStr)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	return pwBytes
}

func getPWFromHexString(pwStr string) []byte {
	pwBytes, err := hex.DecodeString(fmt.Sprintf("%0*d%s",
		66-len(pwStr), 0, pwStr))
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	return pwBytes
}
