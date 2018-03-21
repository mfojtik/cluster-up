package log

var (
	// Logging level (can be set from the CLI).
	// Higher number means more messages.
	LogLevel = 3

	// Default logging backend
	backend = &logrusBackend{}
)

func SetDefaultLogLevel(l int) {
	LogLevel = l
}

var (
	Infof  = backend.Infof
	Error  = backend.Error
	Debugf = backend.Debugf
	Fatal  = backend.Fatal
)

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
