package cmd

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/client/xxdk"
	"gitlab.com/elixxir/primitives/excludedRounds"
	"gitlab.com/xx_network/primitives/id"
	"strconv"
	"strings"
	"time"
)

// Key used for storing xxdk.ReceptionIdentity objects
const IdentityStorageKey = "identityStorageKey"

//////////////////////////////////////////////////////////////////////////////////////
// Parameters
//////////////////////////////////////////////////////////////////////////////////////

func InitParams() (xxdk.CMIXParams, xxdk.E2EParams) {
	e2eParams := xxdk.GetDefaultE2EParams()
	e2eParams.Session.MinKeys = uint16(viper.GetUint(E2eMinKeysFlag))
	e2eParams.Session.MaxKeys = uint16(viper.GetUint(E2eMaxKeysFlag))
	e2eParams.Session.NumRekeys = uint16(viper.GetUint(E2eNumReKeysFlag))
	e2eParams.Session.RekeyThreshold = viper.GetFloat64(E2eRekeyThresholdFlag)

	if viper.GetBool("splitSends") {
		e2eParams.Base.ExcludedRounds = excludedRounds.NewSet()
	}

	cmixParams := xxdk.GetDefaultCMixParams()
	cmixParams.Network.Pickup.ForceHistoricalRounds = viper.GetBool(
		ForceHistoricalRoundsFlag)
	cmixParams.Network.FastPolling = !viper.GetBool(SlowPollingFlag)
	cmixParams.Network.Pickup.ForceMessagePickupRetry = viper.GetBool(
		ForceMessagePickupRetryFlag)
	if cmixParams.Network.Pickup.ForceMessagePickupRetry {
		period := 3 * time.Second
		jww.INFO.Printf("Setting Uncheck Round Period to %v", period)
		cmixParams.Network.Pickup.UncheckRoundPeriod = period
	}
	cmixParams.Network.VerboseRoundTracking = viper.GetBool(
		VerboseRoundTrackingFlag)
	return cmixParams, e2eParams
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

/////////////////////////////////////////////////////////////////////////////////////
// Printing functions
/////////////////////////////////////////////////////////////////////////////////////

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

