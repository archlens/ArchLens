package utils

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Sugar *zap.SugaredLogger

func WithWorkingDirectory(path string) func() {
	currentWd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	err = os.Chdir(path)
	if err != nil {
		panic(err)
	}
	return func() {
		if err := os.Chdir(currentWd); err != nil {
			panic(err)
		}
	}
}

func NewPrettySugaredLogger(verbose, silence bool) *zap.SugaredLogger {
	// Default to Info; flags are mutually exclusive so at most one applies
	level := zap.InfoLevel
	switch {
	case verbose:
		level = zap.DebugLevel
	case silence:
		level = zap.ErrorLevel
	}

	// Customize the encoder config for terminal output
	encoderConfig := zap.NewDevelopmentEncoderConfig()
	
	// Human-readable timestamps (e.g. 2026-10-01T12:00:00.000+0200) instead of epoch numbers
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder 
	
	// Add color to levels (INFO in green, ERROR in red, etc.)
	encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder 
	
	// Shorten caller file paths (e.g., caching/cache.go:101)
	encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder 

	// Use ConsoleEncoder instead of JSONEncoder
	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig),
		zapcore.AddSync(os.Stdout),
		level,
	)

	// Build logger with caller & stacktrace options enabled
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.ErrorLevel))
	
	Sugar = logger.Sugar()

	return Sugar
}