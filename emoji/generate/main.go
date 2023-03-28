////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

// package main downloads the latest list of emojis from Unicode and parses them
// into a Go map that can be used to validate emojis.

package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
)

// Flag variables.
var (
	logLevel int
	logFile  string
	p        = DefaultParams()
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var cmd = &cobra.Command{
	Use: "generateEmojiMap",
	Short: "Downloads the emoji file (from Unicode) and parses them into a " +
		"map that can be saved as a Go file or JSON file.",
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {

		// Initialize the logging
		initLog(jww.Threshold(logLevel), logFile)

		err := generate(p)
		if err != nil {
			jww.FATAL.Panic(err)
		}
	},
}

// init is the initialization function for Cobra which defines flags.
func init() {
	cmd.Flags().StringVarP(&p.DownloadURL, "url", "u", p.DownloadURL,
		"URL to download emojis from.")
	cmd.Flags().StringVarP(&p.GoOutput, "output", "o", p.GoOutput,
		"Output file path for Go file. Set to empty for no output.")
	cmd.Flags().StringVarP(&p.JsonOutput, "json", "j", p.JsonOutput,
		"Output file path for JSON file. Set to empty for no output.")
	cmd.Flags().StringVarP(&p.CodePointDelim, "delim", "d", p.CodePointDelim,
		"The separator used between codepoints.")
	cmd.Flags().StringVarP(&logFile, "log", "l", "-",
		"Log output path. By default, logs are printed to stdout. "+
			"To disable logging, set this to empty (\"\").")
	cmd.Flags().IntVarP(&logLevel, "logLevel", "v", 4,
		"Verbosity level of logging. 0 = TRACE, 1 = DEBUG, 2 = INFO, "+
			"3 = WARN, 4 = ERROR, 5 = CRITICAL, 6 = FATAL")
}

// initLog will enable JWW logging to the given log path with the given
// threshold. If log path is empty, then logging is not enabled. Panics if the
// log file cannot be opened or if the threshold is invalid.
func initLog(threshold jww.Threshold, logPath string) {
	if logPath == "" {
		// Do not enable logging if no log file is set
		return
	} else if logPath != "-" {
		// Set the log file if stdout is not selected

		// Disable stdout output
		jww.SetStdoutOutput(io.Discard)

		// Use log file
		logOutput, err :=
			os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
		jww.SetLogOutput(logOutput)
	}

	if threshold < jww.LevelTrace || threshold > jww.LevelFatal {
		panic("Invalid log threshold: " + strconv.Itoa(int(threshold)))
	}

	// Display microseconds if the threshold is set to TRACE or DEBUG
	if threshold == jww.LevelTrace || threshold == jww.LevelDebug {
		jww.SetFlags(log.LstdFlags | log.Lmicroseconds)
	}

	// Enable logging
	jww.SetStdoutThreshold(threshold)
	jww.SetLogThreshold(threshold)
	jww.INFO.Printf("Log level set to: %s", threshold)
}
