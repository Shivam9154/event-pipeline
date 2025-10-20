package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

var Log *logrus.Logger

func init() {
	Log = logrus.New()
	Log.SetOutput(os.Stdout)
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
