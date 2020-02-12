////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package globals

import (
	jww "github.com/spf13/jwalterweatherman"
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
	Log = jww.NewNotepad(jww.LevelError, jww.LevelInfo, os.Stdout,
		ioutil.Discard, "CLIENT", log.Ldate|log.Ltime)
	// If verbose flag set then log more info for debugging
	if verbose {
		Log.SetLogThreshold(jww.LevelDebug)
		Log.SetStdoutThreshold(jww.LevelDebug)
		Log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	} else {
		Log.SetLogThreshold(jww.LevelInfo)
		Log.SetStdoutThreshold(jww.LevelInfo)
	}
	if logPath != "" || logPath == "-" {
		// Create log file, overwrites if existing
		logFile, err := os.Create(logPath)
		if err != nil {
			Log.WARN.Println("Invalid or missing log path," +
				" default path used.")
		} else {
			Log.SetLogOutput(logFile)
		}
	}
}
