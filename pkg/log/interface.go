package log

type Logger interface {
	// Infof is informative message
	Infof(format string, args ...interface{})

	// Fatal is fatal (os.Exit(1))
	Fatal(err error)

	// Errorf is for reporting non-fatal errors. Can be used as an
	// error wrapper when returning errors.
	Errorf(message string, err error) error

	// Debugf is for debugging...
	Debugf(format string, args ...interface{})
}
