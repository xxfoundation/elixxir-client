package cmd

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/client/xxdk"
	"gitlab.com/elixxir/primitives/excludedRounds"
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
