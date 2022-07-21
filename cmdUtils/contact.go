package cmd

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/xx_network/primitives/id"
	"io/ioutil"
	"strconv"
	"strings"
)

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
	outfilePath := viper.GetString(WriteContactFlag)
	if outfilePath == "" {
		return
	}
	jww.INFO.Printf("PubKey WRITE: %s", c.DhPubKey.Text(10))
	err := ioutil.WriteFile(outfilePath, c.Marshal(), 0644)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
}

func PrintContact(c contact.Contact) {
	jww.DEBUG.Printf("Printing contact: %+v", c)
	cBytes := c.Marshal()
	if len(cBytes) == 0 {
		jww.ERROR.Print("Marshaled contact has a size of 0.")
	} else {
		jww.DEBUG.Printf("Printing marshaled contact of size %d.", len(cBytes))
	}

	jww.INFO.Printf(string(cBytes))
}

func ParseRecipient(idStr string) *id.ID {
	if idStr == "0" {
		jww.FATAL.Panicf("No recipient specified")
	}

	if strings.HasPrefix(idStr, "0x") {
		return getUIDFromHexString(idStr[2:])
	} else if strings.HasPrefix(idStr, "b64:") {
		return getUIDFromb64String(idStr[4:])
	} else {
		return getUIDFromString(idStr)
	}
}

func getUIDFromHexString(idStr string) *id.ID {
	idBytes, err := hex.DecodeString(fmt.Sprintf("%0*d%s",
		66-len(idStr), 0, idStr))
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	ID, err := id.Unmarshal(idBytes)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	return ID
}

func getUIDFromb64String(idStr string) *id.ID {
	idBytes, err := base64.StdEncoding.DecodeString(idStr)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	ID, err := id.Unmarshal(idBytes)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	return ID
}

func getUIDFromString(idStr string) *id.ID {
	idInt, err := strconv.Atoi(idStr)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	if idInt > 255 {
		jww.FATAL.Panicf("cannot convert integers above 255. Use 0x " +
			"or b64: representation")
	}
	idBytes := make([]byte, 33)
	binary.BigEndian.PutUint64(idBytes, uint64(idInt))
	idBytes[32] = byte(id.User)
	ID, err := id.Unmarshal(idBytes)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
	return ID
}
