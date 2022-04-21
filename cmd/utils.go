package cmd

import (
	"io/ioutil"
	"strconv"
	"strings"

	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/client/cmix"
	backupCrypto "gitlab.com/elixxir/crypto/backup"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/utils"
)

// todo: go through cmd package and organize utility functions

func loadBackup(backupPath, backupPass string) (backupCrypto.Backup, []byte) {
	jww.INFO.Printf("Loading backup from path %q with password %q", backupPath, backupPass)
	backupFile, err := utils.ReadFile(backupPath)
	if err != nil {
		jww.FATAL.Panicf("%v", err)
	}

	var b backupCrypto.Backup
	err = b.Decrypt(backupPass, backupFile)
	if err != nil {
		jww.ERROR.Printf("Failed to decrypt backup: %+v", err)
	}

	return b, backupFile
}

/////////////////////////////////////////////////////////////////
////////////////// Print functions /////////////////////////////
/////////////////////////////////////////////////////////////////

// Helper function which prints the round resuls
func printRoundResults(allRoundsSucceeded, timedOut bool,
	rounds map[id.Round]cmix.RoundResult, roundIDs []id.Round,
	payload []byte, recipient *id.ID) {

	// Done as string slices for easy and human readable printing
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
		jww.ERROR.Printf("\tRound(s) %v timed "+
			"\n\tout (no network resolution could be found)", strings.Join(timedOutRounds, ","))
	}

}

func writeContact(c contact.Contact) {
	outfilePath := viper.GetString("writeContact")
	if outfilePath == "" {
		return
	}
	jww.INFO.Printf("PubKey WRITE: %s", c.DhPubKey.Text(10))
	err := ioutil.WriteFile(outfilePath, c.Marshal(), 0644)
	if err != nil {
		jww.FATAL.Panicf("%+v", err)
	}
}

func readContact() contact.Contact {
	inputFilePath := viper.GetString("destfile")
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
	jww.INFO.Printf("PubKey READ: %s", c.DhPubKey.Text(10))
	return c
}
