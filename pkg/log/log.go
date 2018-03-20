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
