package logger

import "github.com/sirupsen/logrus"

// NewLogger is a default logger
func NewLogger() *logrus.Logger {
	log := logrus.New()
	log.SetLevel(logrus.DebugLevel)
	return log
}
