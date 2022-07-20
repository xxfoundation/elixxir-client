package cmd

import (
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/client/cmix"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/xx_network/primitives/id"
	"io/ioutil"
)

// todo: go through cmd package and organize utility functions

func printContact(c contact.Contact) {
	jww.DEBUG.Printf("Printing contact: %+v", c)
	cBytes := c.Marshal()
	if len(cBytes) == 0 {
		jww.ERROR.Print("Marshaled contact has a size of 0.")
	} else {
		jww.DEBUG.Printf("Printing marshaled contact of size %d.", len(cBytes))
	}

	jww.INFO.Printf(string(cBytes))
}

func writeContact(c contact.Contact) {
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

func readContact(inputFilePath string) contact.Contact {
	if inputFilePath == "" {
		return contact.Contact{}
	}

	data, err := ioutil.ReadFile(inputFilePath)
	jww.INFO.Printf("Contact file size read in: %d", len(data))
	if err != nil {
		jww.FATAL.Panicf("Failed to read contact file: %+v", err)
	}
	c, err := contact.Unmarshal(data)
	if err != nil {
		jww.FATAL.Panicf("Failed to unmarshal contact: %+v", err)
	}
	jww.INFO.Printf("CONTACTPUBKEY READ: %s",
		c.DhPubKey.TextVerbose(16, 0))
	jww.INFO.Printf("Contact ID: %s", c.ID)
	return c
}

func makeVerifySendsCallback(retryChan, done chan struct{}) cmix.RoundEventCallback {
	return func(allRoundsSucceeded, timedOut bool, rounds map[id.Round]cmix.RoundResult) {
		if !allRoundsSucceeded {
			retryChan <- struct{}{}
		} else {
			done <- struct{}{}
		}
	}
}
