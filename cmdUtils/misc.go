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

// BindPersistentFlagHelper binds the key to a Persistent pflag.Flag used by Cobra and prints an
// error if one occurs.
func BindPersistentFlagHelper(key string, command *cobra.Command) {
	err := viper.BindPFlag(key, command.PersistentFlags().Lookup(key))
	if err != nil {
		jww.ERROR.Printf("viper.BindPFlag failed for %q: %+v", key, err)
	}
}

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
