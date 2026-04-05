package logger

import (
	"log/slog"
	"os"
	"time"

	"github.com/rs/zerolog"
	slogzerolog "github.com/samber/slog-zerolog/v2"
)

// Setup initialises zerolog and registers it as the default slog handler.
// Call once at the start of main(). All slog.Info/Error/Debug calls then go through zerolog.
func Setup(debug bool) {
	zerolog.TimeFieldFormat = time.RFC3339

	level := zerolog.InfoLevel
	slogLevel := slog.LevelInfo
	if debug {
		level = zerolog.DebugLevel
		slogLevel = slog.LevelDebug
	}

	zl := zerolog.New(
		zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339},
	).Level(level).With().Timestamp().Logger()

	handler := slogzerolog.Option{
		Level:  slogLevel,
		Logger: &zl,
	}.NewZerologHandler()

	slog.SetDefault(slog.New(handler))
}
