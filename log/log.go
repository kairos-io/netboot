package log

import (
	"github.com/rs/zerolog"
	"time"
)

var Log zerolog.Logger

// SetDefaultLogger to stdout with timestamp
func SetDefaultLogger() {
	Log = zerolog.New(zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
		w.TimeFormat = time.RFC3339
	})).With().Timestamp().Logger()
}
