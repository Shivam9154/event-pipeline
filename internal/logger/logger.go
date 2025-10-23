package logger

import (
	"io"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

var Log *logrus.Logger

func init() {
	Log = logrus.New()

	// Single log file with 24-hour expiry semantics
	const logFile = "app.log"

	// If the log file exists and is older than 24 hours, reset it
	if fi, err := os.Stat(logFile); err == nil {
		if time.Since(fi.ModTime()) > 24*time.Hour {
			_ = os.Remove(logFile)
		}
	}

	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err == nil {
		// Write to both stdout and the file so logs remain visible in terminal
		Log.SetOutput(io.MultiWriter(os.Stdout, f))
	} else {
		// Fallback to stdout if file can't be opened
		Log.SetOutput(os.Stdout)
	}

	Log.SetFormatter(&logrus.JSONFormatter{})
	Log.SetLevel(logrus.InfoLevel)
}

// WithEventID returns a logger with eventId field
func WithEventID(eventID string) *logrus.Entry {
	return Log.WithField("eventId", eventID)
}

// WithFields returns a logger with custom fields
func WithFields(fields logrus.Fields) *logrus.Entry {
	return Log.WithFields(fields)
}
