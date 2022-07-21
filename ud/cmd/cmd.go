package cmd

import (
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	cmdUtils "gitlab.com/elixxir/client/cmdUtils"
	"gitlab.com/elixxir/client/single"
	"gitlab.com/elixxir/client/ud"
	"gitlab.com/elixxir/client/xxmutils"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/primitives/utils"
	"strings"
	"time"
)

func Start() {
	// Initialize paramaters
	cmixParams, e2eParams := cmdUtils.InitParams()

	// Initialize log
	cmdUtils.InitLog(viper.GetUint(cmdUtils.LogLevelFlag), viper.GetString(cmdUtils.LogFlag))

	// Initialize messenger
	authCbs := cmdUtils.MakeAuthCallbacks(
		viper.GetBool(cmdUtils.UnsafeChannelCreationFlag), e2eParams)
	messenger := cmdUtils.InitE2e(cmixParams, e2eParams, authCbs)

	// get user and save contact to file
	user := messenger.GetReceptionIdentity()
	jww.INFO.Printf("[UD]User: %s", user.ID)
	cmdUtils.WriteContact(user.GetContact())

	err := messenger.StartNetworkFollower(50 * time.Millisecond)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}

	jww.TRACE.Printf("[UD] Waiting for connection...")

	// Wait until connected or crash on timeout
	connected := make(chan bool, 10)
	messenger.GetCmix().AddHealthCallback(
		func(isconnected bool) {
			connected <- isconnected
		})
	cmdUtils.WaitUntilConnected(connected)

	jww.TRACE.Printf("[UD] Connected!")

	// Make user discovery manager
	rng := messenger.GetRng()
	userToRegister := viper.GetString(UdRegisterFlag)
	jww.TRACE.Printf("[UD] Registering user %v...", userToRegister)
	userDiscoveryMgr, err := ud.NewManager(messenger, messenger.GetComms(),
		messenger.NetworkFollowerStatus, userToRegister, nil)
	if err != nil {
		// todo: probably better to have a newOrLoadManager for
		if strings.Contains(err.Error(), ud.IsRegisteredErr) {
			userDiscoveryMgr, err = ud.LoadManager(messenger, messenger.GetComms())
			if err != nil {
				jww.FATAL.Panicf("Failed to load UD manager: %+v", err)
			}
		} else {
			jww.FATAL.Panicf("Failed to create new UD manager: %+v", err)

		}
	}
	jww.INFO.Printf("[UD] Registered user %v", userToRegister)

	var newFacts fact.FactList
	phone := viper.GetString(UdAddPhoneFlag)
	if phone != "" {
		f, err := fact.NewFact(fact.Phone, phone)
		if err != nil {
			jww.FATAL.Panicf("Failed to create new fact: %+v", err)
		}
		newFacts = append(newFacts, f)
	}

	email := viper.GetString(UdAddEmailFlag)
	if email != "" {
		f, err := fact.NewFact(fact.Email, email)
		if err != nil {
			jww.FATAL.Panicf("Failed to create new fact: %+v", err)
		}
		newFacts = append(newFacts, f)
	}

	for i := 0; i < len(newFacts); i++ {
		jww.INFO.Printf("[UD] Registering Fact: %v",
			newFacts[i])
		r, err := userDiscoveryMgr.SendRegisterFact(newFacts[i])
		if err != nil {
			fmt.Printf("Failed to register fact: %s\n",
				newFacts[i])
			jww.FATAL.Panicf("[UD] Failed to send register fact: %+v", err)
		}
		// TODO Store the code?
		jww.INFO.Printf("[UD] Fact Add Response: %+v", r)
	}

	confirmID := viper.GetString(UdConfirmFlag)
	if confirmID != "" {
		jww.INFO.Printf("[UD] Confirming fact: %v", confirmID)
		err = userDiscoveryMgr.ConfirmFact(confirmID, confirmID)
		if err != nil {
			fmt.Printf("Couldn't confirm fact: %s\n",
				err.Error())
			jww.FATAL.Panicf("%+v", err)
		}

		jww.INFO.Printf("[UD] Confirmed %v", confirmID)
	}

	udContact, err := userDiscoveryMgr.GetContact()
	if err != nil {
		fmt.Printf("Failed to get user discovery contact object: %+v", err)
		jww.FATAL.Printf("Failed to get user discovery contact object: %+v", err)
	}

	// Handle lookup (verification) process
	// Note: Cryptographic verification occurs above the bindings layer
	lookupIDStr := viper.GetString(UdLookupFlag)
	if lookupIDStr != "" {
		lookupID := cmdUtils.ParseRecipient(lookupIDStr)
		jww.INFO.Printf("[UD] Looking up %v", lookupID)

		cb := func(newContact contact.Contact, err error) {
			if err != nil {
				jww.FATAL.Panicf("UserDiscovery Lookup error: %+v", err)
			}
			cmdUtils.PrintContact(newContact)
		}

		stream := rng.GetStream()
		_, _, err = ud.Lookup(messenger.GetCmix(),
			stream, messenger.GetE2E().GetGroup(),
			udContact, cb, lookupID, single.GetDefaultRequestParams())
		if err != nil {
			jww.WARN.Printf("Failed UD lookup: %+v", err)
		}
		stream.Close()

		time.Sleep(31 * time.Second)
	}

	if viper.IsSet(UdBatchAddFlag) {
		idListFile, err := utils.ReadFile(viper.GetString(UdBatchAddFlag))
		if err != nil {
			fmt.Printf("BATCHADD: Couldn't read file: %s\n",
				err.Error())
			jww.FATAL.Panicf("BATCHADD: Couldn't read file: %+v", err)
		}
		jww.INFO.Printf("[UD] BATCHADD: Running")
		restored, _, _, err := xxmutils.RestoreContactsFromBackup(
			idListFile, messenger, userDiscoveryMgr, nil)
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}
		for i := 0; i < len(restored); i++ {
			uid := restored[i]
			for !messenger.GetE2E().HasAuthenticatedChannel(uid) {
				time.Sleep(time.Second)
			}
			jww.INFO.Printf("[UD] Authenticated channel established for %s", uid)
		}
	}
	usernameSearchStr := viper.GetString(UdSearchUsernameFlag)
	emailSearchStr := viper.GetString(UdSearchEmailFlag)
	phoneSearchStr := viper.GetString(UdSearchPhoneFlag)

	var facts fact.FactList
	if usernameSearchStr != "" {
		f, err := fact.NewFact(fact.Username, usernameSearchStr)
		if err != nil {
			jww.FATAL.Panicf("Failed to create new fact: %+v", err)
		}
		facts = append(facts, f)
	}
	if emailSearchStr != "" {
		f, err := fact.NewFact(fact.Email, emailSearchStr)
		if err != nil {
			jww.FATAL.Panicf("Failed to create new fact: %+v", err)
		}
		facts = append(facts, f)
	}
	if phoneSearchStr != "" {
		f, err := fact.NewFact(fact.Phone, phoneSearchStr)
		if err != nil {
			jww.FATAL.Panicf("Failed to create new fact: %+v", err)
		}
		facts = append(facts, f)
	}

	userToRemove := viper.GetString(UdRemoveFlag)
	if userToRemove != "" {
		f, err := fact.NewFact(fact.Username, userToRemove)
		if err != nil {
			jww.FATAL.Panicf(
				"Failed to create new fact: %+v", err)
		}
		err = userDiscoveryMgr.PermanentDeleteAccount(f)
		if err != nil {
			fmt.Printf("Couldn't remove user %s\n",
				userToRemove)
			jww.FATAL.Panicf(
				"Failed to remove user %s: %+v",
				userToRemove, err)
		}
		fmt.Printf("Removed user from discovery: %s\n",
			userToRemove)
	}

	if len(facts) == 0 {
		err = messenger.StopNetworkFollower()
		if err != nil {
			jww.WARN.Print(err)
		}
		return
	}

	cb := func(contacts []contact.Contact, err error) {
		if err != nil {
			jww.FATAL.Panicf("%+v", err)
		}
		for _, c := range contacts {
			cmdUtils.PrintContact(c)
		}
	}

	stream := rng.GetStream()
	defer stream.Close()
	jww.INFO.Printf("[UD] Search: %v", facts)
	// todo: for Search & Lookup, consider passing messenger and pulling the fields internally
	_, _, err = ud.Search(messenger.GetCmix(),
		messenger.GetEventReporter(),
		stream, messenger.GetE2E().GetGroup(),
		udContact, cb, facts, single.GetDefaultRequestParams())
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}

	time.Sleep(91 * time.Second)
	err = messenger.StopNetworkFollower()
	if err != nil {
		jww.WARN.Print(err)
	}

}
