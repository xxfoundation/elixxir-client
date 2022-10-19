////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

// logging.go contains bindings log control functions

package bindings

import (
	//"log"

	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"google.golang.org/grpc/grpclog"
)

// LogLevel sets level of logging. All logs at the set level and below will be
// displayed (e.g., when log level is ERROR, only ERROR, CRITICAL, and FATAL
// messages will be printed).
//
// Log level options:
//	TRACE    - 0
//	DEBUG    - 1
//	INFO     - 2
//	WARN     - 3
//	ERROR    - 4
//	CRITICAL - 5
//	FATAL    - 6
//
// The default log level without updates is INFO.
func LogLevel(level int) error {
	if level < 0 || level > 6 {
		return errors.Errorf("log level is not valid: log level: %d", level)
	}

	threshold := jww.Threshold(level)
	jww.SetLogThreshold(threshold)
	jww.SetStdoutThreshold(threshold)
	//jww.SetFlags(log.LstdFlags | log.Lmicroseconds)

	switch threshold {
	case jww.LevelTrace:
		fallthrough
	case jww.LevelDebug:
		fallthrough
	case jww.LevelInfo:
		jww.INFO.Printf("Log level set to: %s", threshold)
	case jww.LevelWarn:
		jww.WARN.Printf("Log level set to: %s", threshold)
	case jww.LevelError:
		jww.ERROR.Printf("Log level set to: %s", threshold)
	case jww.LevelCritical:
		jww.CRITICAL.Printf("Log level set to: %s", threshold)
	case jww.LevelFatal:
		jww.FATAL.Printf("Log level set to: %s", threshold)
	}

	return nil
}

type LogWriter interface {
	Log(string)
}

// RegisterLogWriter registers a callback on which logs are written.
func RegisterLogWriter(writer LogWriter) {
	jww.SetLogOutput(&writerAdapter{lw: writer})
}

// EnableGrpcLogs sets GRPC trace logging.
func EnableGrpcLogs(writer LogWriter) {
	logger := &writerAdapter{lw: writer}
	grpclog.SetLoggerV2(grpclog.NewLoggerV2WithVerbosity(
		logger, logger, logger, 99))
}

type writerAdapter struct {
	lw LogWriter
}

func (wa *writerAdapter) Write(p []byte) (n int, err error) {
	wa.lw.Log(string(p))
	return len(p), nil
}
