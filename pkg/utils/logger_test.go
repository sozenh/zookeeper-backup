package utils

import (
	"testing"

	"go.uber.org/zap"
)

func TestInitLogger(t *testing.T) {
	tests := []struct {
		name   string
		level  string
		format string
	}{
		{
			name:   "json format info level",
			level:  "info",
			format: "json",
		},
		{
			name:   "text format debug level",
			level:  "debug",
			format: "text",
		},
		{
			name:   "text format error level",
			level:  "error",
			format: "text",
		},
		{
			name:   "invalid level defaults to info",
			level:  "invalid",
			format: "text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := InitLogger(tt.level, tt.format)
			if err != nil {
				t.Fatalf("InitLogger() error = %v", err)
			}

			logger := GetLogger()
			if logger == nil {
				t.Error("GetLogger() should return non-nil logger after InitLogger")
			}
		})
	}
}

func TestGetLogger(t *testing.T) {
	t.Run("get logger without init", func(t *testing.T) {
		// Reset global logger
		globalLogger = nil

		logger := GetLogger()
		if logger == nil {
			t.Error("GetLogger() should return fallback logger when not initialized")
		}
	})

	t.Run("get logger after init", func(t *testing.T) {
		InitLogger("info", "text")

		logger := GetLogger()
		if logger == nil {
			t.Error("GetLogger() should return initialized logger")
		}
	})
}

func TestLoggingFunctions(t *testing.T) {
	// Initialize logger for tests
	InitLogger("debug", "text")

	t.Run("info logging", func(t *testing.T) {
		// Should not panic
		Info("test info message", zap.String("key", "value"))
	})

	t.Run("error logging", func(t *testing.T) {
		// Should not panic
		Error("test error message", zap.String("key", "value"))
	})

	t.Run("warn logging", func(t *testing.T) {
		// Should not panic
		Warn("test warn message", zap.String("key", "value"))
	})

	t.Run("debug logging", func(t *testing.T) {
		// Should not panic
		Debug("test debug message", zap.String("key", "value"))
	})
}

func TestSync(t *testing.T) {
	t.Run("sync with initialized logger", func(t *testing.T) {
		InitLogger("info", "text")
		// Should not panic
		Sync()
	})

	t.Run("sync without initialized logger", func(t *testing.T) {
		globalLogger = nil
		// Should not panic
		Sync()
	})
}
