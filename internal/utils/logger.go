package utils

import (
	"io"

	"github.com/apex/log"
	"github.com/apex/log/handlers/logfmt"
)

func NewLogger(level log.Level, debug bool, output io.Writer) *log.Logger {
	logger := &log.Logger{}

	if debug {
		logger.Level = log.DebugLevel
	} else {
		logger.Level = level
	}

	logger.Handler = logfmt.New(output)

	return logger
}
