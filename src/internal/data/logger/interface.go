package logger

// Logger defines the interface for the application's logger.
type Logger interface {
	Printf(format string, v ...any)
	Fatalf(format string, v ...any)
	Println(v ...any)
	Close()
}
