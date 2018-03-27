package log

import (
	"os"

	"github.com/Sirupsen/logrus"
)

func init() {
	// Log as JSON instead of the default ASCII formatter.
	logrus.SetFormatter(&logrus.TextFormatter{})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	logrus.SetOutput(os.Stdout)

	// Only log the warning severity or above.
	logrus.SetLevel(logrus.InfoLevel)
}

type logrusBackend struct {
	componentName string
}

func (b *logrusBackend) logger() *logrus.Logger {
	if len(b.componentName) == 0 {
		return logrus.StandardLogger()
	}
	return logrus.WithFields(logrus.Fields{"component": b.componentName}).Logger
}

func (b *logrusBackend) Infof(format string, args ...interface{}) {
	b.logger().Infof(format, args...)
}

func (b *logrusBackend) Fatal(err error) {
	b.logger().Fatal(err.Error())
}

func (b *logrusBackend) Error(message string, err error) error {
	if LogLevel >= 5 {
		b.logger().WithError(err).Error(message)
	}
	return err
}

func (b *logrusBackend) Debugf(format string, args ...interface{}) {
	if LogLevel <= 3 {
		return
	}
	loggerCopy := b.logger()
	loggerCopy.SetLevel(logrus.DebugLevel)
	loggerCopy.Debugf(format, args...)
}
