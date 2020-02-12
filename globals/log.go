////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package globals

import (
	jww "github.com/spf13/jwalterweatherman"
	"io"
	"io/ioutil"
	"log"
	"os"
)

// Log is logging everything to this notepad so that the CUI can replace it
// with its own notepad and get logging statements from the client
var Log = jww.NewNotepad(jww.LevelInfo, jww.LevelInfo, os.Stdout,
	ioutil.Discard, "CLIENT", log.Ldate|log.Ltime)

// InitLog initializes logging thresholds and the log path.
// verbose turns on debug logging, setting the log path to nil
// uses std out.
func InitLog(verbose bool, logPath string) {
	logLevel := jww.LevelInfo
	logFlags := (log.Ldate | log.Ltime)
	stdOut := io.Writer(os.Stdout)
	logFile := ioutil.Discard

	// If the verbose flag is set, print all logs and
	// print microseconds as well
	if verbose {
		logLevel = jww.LevelDebug
		logFlags = (log.Ldate | log.Ltime | log.Lmicroseconds)
	}
	// If the logpath is empty or not set to - (stdout),
	// set up the log file and do not log to stdout
	if logPath != "" && logPath != "-" {
		// Create log file, overwrites if existing
		lF, err := os.Create(logPath)
		if err != nil {
			jww.WARN.Println("Invalid or missing log path," +
				" stdout used.")
		} else {
			logFile = io.Writer(lF)
			stdOut = ioutil.Discard
		}
	}

	Log = jww.NewNotepad(logLevel, logLevel, stdOut, logFile,
		"CLIENT", logFlags)
}
