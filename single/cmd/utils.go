package cmd

import (
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/xx_network/primitives/utils"
)

// readSingleUseContact opens the contact specified in the CLI flags. Panics if
// no file provided or if an error occurs while reading or unmarshalling it.
func readSingleUseContact(key string) contact.Contact {
	// get path
	filePath := viper.GetString(key)
	if filePath == "" {
		jww.FATAL.Panicf("Failed to read contact file: no file path provided.")
	}

	// Read from file
	data, err := utils.ReadFile(filePath)
	jww.INFO.Printf("Contact file size read in: %d bytes", len(data))
	if err != nil {
		jww.FATAL.Panicf("Failed to read contact file: %+v", err)
	}

	// Unmarshal contact
	c, err := contact.Unmarshal(data)
	if err != nil {
		jww.FATAL.Panicf("Failed to unmarshal contact: %+v", err)
	}

	return c
}
