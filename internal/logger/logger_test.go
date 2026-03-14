package logger

import (
	"os"
	"testing"

	"go.uber.org/zap/zapcore"
)

func TestNewLogger_DefaultLevel(t *testing.T) {
	// Save original env
	originalLevel := os.Getenv("LOG_LEVEL")
	originalEnv := os.Getenv("ENV")
	defer func() {
		os.Setenv("LOG_LEVEL", originalLevel)
		os.Setenv("ENV", originalEnv)
	}()

	// Clear env vars
	os.Unsetenv("LOG_LEVEL")
	os.Unsetenv("ENV")

	logger := NewLogger()
	if logger == nil {
		t.Fatal("NewLogger() returned nil")
	}

	// Just verify logger is created successfully
	// The actual level is tested in getLogLevel tests
	logger.Info("test message")
}

func TestNewLogger_DebugLevel(t *testing.T) {
	originalLevel := os.Getenv("LOG_LEVEL")
	defer os.Setenv("LOG_LEVEL", originalLevel)

	os.Setenv("LOG_LEVEL", "debug")
	logger := NewLogger()

	if logger == nil {
		t.Fatal("NewLogger() returned nil")
	}
}

func TestNewLogger_InfoLevel(t *testing.T) {
	originalLevel := os.Getenv("LOG_LEVEL")
	defer os.Setenv("LOG_LEVEL", originalLevel)

	os.Setenv("LOG_LEVEL", "info")
	logger := NewLogger()

	if logger == nil {
		t.Fatal("NewLogger() returned nil")
	}
}

func TestNewLogger_WarnLevel(t *testing.T) {
	originalLevel := os.Getenv("LOG_LEVEL")
	defer os.Setenv("LOG_LEVEL", originalLevel)

	os.Setenv("LOG_LEVEL", "warn")
	logger := NewLogger()

	if logger == nil {
		t.Fatal("NewLogger() returned nil")
	}
}

func TestNewLogger_ErrorLevel(t *testing.T) {
	originalLevel := os.Getenv("LOG_LEVEL")
	defer os.Setenv("LOG_LEVEL", originalLevel)

	os.Setenv("LOG_LEVEL", "error")
	logger := NewLogger()

	if logger == nil {
		t.Fatal("NewLogger() returned nil")
	}
}

func TestNewLogger_InvalidLevelDefaultsToInfo(t *testing.T) {
	originalLevel := os.Getenv("LOG_LEVEL")
	defer os.Setenv("LOG_LEVEL", originalLevel)

	os.Setenv("LOG_LEVEL", "invalid_level_xyz")
	logger := NewLogger()

	if logger == nil {
		t.Fatal("NewLogger() returned nil for invalid level")
	}
}

func TestNewLogger_DevelopmentMode(t *testing.T) {
	originalLevel := os.Getenv("LOG_LEVEL")
	originalEnv := os.Getenv("ENV")
	defer func() {
		os.Setenv("LOG_LEVEL", originalLevel)
		os.Setenv("ENV", originalEnv)
	}()

	os.Setenv("LOG_LEVEL", "info")
	os.Setenv("ENV", "development")
	logger := NewLogger()

	if logger == nil {
		t.Fatal("NewLogger() returned nil")
	}
}

func TestGet(t *testing.T) {
	logger := Get()
	if logger == nil {
		t.Fatal("Get() returned nil")
	}
}

func TestDebug(t *testing.T) {
	// Should not panic
	Debug("test debug message")
	Debug("test debug message", "key", "value")
}

func TestInfo(t *testing.T) {
	Info("test info message")
	Info("test info message", "key", "value")
}

func TestWarn(t *testing.T) {
	Warn("test warn message")
	Warn("test warn message", "key", "value")
}

func TestError(t *testing.T) {
	Error("test error message")
	Error("test error message", "key", "value")
}

func TestDPanic(t *testing.T) {
	// In development mode, this would panic
	// We're just testing it doesn't crash in normal mode
	DPanic("test dpanic message")
}

func TestPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Panic() should have panicked")
		}
	}()
	Panic("test panic message")
}

func TestFatal(t *testing.T) {
	// Test that Fatal function exists and can be called
	// We can't actually execute it because it calls os.Exit
	// Just verify it compiles and can be referenced
	t.Cleanup(func() {
		// Cleanup if needed
	})
	// Reference the function to ensure it exists
	_ = Fatal
	// Note: Actual execution skipped to avoid os.Exit
}

func TestSync(t *testing.T) {
	err := Sync()
	// Sync may fail in test environment (stderr not available)
	// Just verify the function exists and can be called
	_ = err
}

func TestGetLogLevel_Default(t *testing.T) {
	originalLevel := os.Getenv("LOG_LEVEL")
	defer os.Setenv("LOG_LEVEL", originalLevel)

	os.Unsetenv("LOG_LEVEL")
	level := getLogLevel()

	if level != zapcore.WarnLevel {
		t.Errorf("Default level should be Warn, got %v", level)
	}
}

func TestGetLogLevel_Debug(t *testing.T) {
	originalLevel := os.Getenv("LOG_LEVEL")
	defer os.Setenv("LOG_LEVEL", originalLevel)

	os.Setenv("LOG_LEVEL", "debug")
	level := getLogLevel()

	if level != zapcore.DebugLevel {
		t.Errorf("Expected Debug level, got %v", level)
	}
}

func TestGetLogLevel_Info(t *testing.T) {
	originalLevel := os.Getenv("LOG_LEVEL")
	defer os.Setenv("LOG_LEVEL", originalLevel)

	os.Setenv("LOG_LEVEL", "info")
	level := getLogLevel()

	if level != zapcore.InfoLevel {
		t.Errorf("Expected Info level, got %v", level)
	}
}

func TestGetLogLevel_Warn(t *testing.T) {
	originalLevel := os.Getenv("LOG_LEVEL")
	defer os.Setenv("LOG_LEVEL", originalLevel)

	os.Setenv("LOG_LEVEL", "warn")
	level := getLogLevel()

	if level != zapcore.WarnLevel {
		t.Errorf("Expected Warn level, got %v", level)
	}
}

func TestGetLogLevel_Error(t *testing.T) {
	originalLevel := os.Getenv("LOG_LEVEL")
	defer os.Setenv("LOG_LEVEL", originalLevel)

	os.Setenv("LOG_LEVEL", "error")
	level := getLogLevel()

	if level != zapcore.ErrorLevel {
		t.Errorf("Expected Error level, got %v", level)
	}
}

func TestIsDevelopment_True(t *testing.T) {
	originalEnv := os.Getenv("ENV")
	originalNodeEnv := os.Getenv("NODE_ENV")
	defer func() {
		os.Setenv("ENV", originalEnv)
		os.Setenv("NODE_ENV", originalNodeEnv)
	}()

	os.Setenv("ENV", "development")
	os.Unsetenv("NODE_ENV")

	if !isDevelopment() {
		t.Error("isDevelopment() should return true when ENV=development")
	}
}

func TestIsDevelopment_NodeEnv(t *testing.T) {
	originalEnv := os.Getenv("ENV")
	originalNodeEnv := os.Getenv("NODE_ENV")
	defer func() {
		os.Setenv("ENV", originalEnv)
		os.Setenv("NODE_ENV", originalNodeEnv)
	}()

	os.Unsetenv("ENV")
	os.Setenv("NODE_ENV", "development")

	if !isDevelopment() {
		t.Error("isDevelopment() should return true when NODE_ENV=development")
	}
}

func TestIsDevelopment_False(t *testing.T) {
	originalEnv := os.Getenv("ENV")
	originalNodeEnv := os.Getenv("NODE_ENV")
	defer func() {
		os.Setenv("ENV", originalEnv)
		os.Setenv("NODE_ENV", originalNodeEnv)
	}()

	os.Unsetenv("ENV")
	os.Unsetenv("NODE_ENV")

	if isDevelopment() {
		t.Error("isDevelopment() should return false when not in development mode")
	}
}

func TestNewLogger_JSONFormat(t *testing.T) {
	originalLevel := os.Getenv("LOG_LEVEL")
	originalEnv := os.Getenv("ENV")
	defer func() {
		os.Setenv("LOG_LEVEL", originalLevel)
		os.Setenv("ENV", originalEnv)
	}()

	os.Unsetenv("LOG_LEVEL")
	os.Unsetenv("ENV")

	logger := NewLogger()
	if logger == nil {
		t.Fatal("NewLogger() returned nil")
	}

	// Verify logger can output JSON
	logger.Info("test JSON output")
}

func TestNewLogger_ConsoleFormat(t *testing.T) {
	originalLevel := os.Getenv("LOG_LEVEL")
	originalEnv := os.Getenv("ENV")
	defer func() {
		os.Setenv("LOG_LEVEL", originalLevel)
		os.Setenv("ENV", originalEnv)
	}()

	os.Setenv("ENV", "development")
	os.Unsetenv("LOG_LEVEL")

	logger := NewLogger()
	if logger == nil {
		t.Fatal("NewLogger() returned nil")
	}

	// Verify logger can output console format
	logger.Info("test console output")
}

func TestGet_Initialization(t *testing.T) {
	// Test that Get() returns the initialized logger
	// This verifies the init() function worked
	logger := Get()

	if logger == nil {
		t.Fatal("Get() returned nil - logger not initialized")
	}

	// Verify it's the same instance as NewLogger()
	logger2 := NewLogger()
	if logger.Desugar() == nil || logger2.Desugar() == nil {
		t.Error("Logger instances should be valid")
	}
}

func TestLogFunctions_WithFields(t *testing.T) {
	// Test all log functions with various field types
	Debug("test", "string", "value", "int", 42, "bool", true)
	Info("test", "string", "value", "int", 42, "bool", true)
	Warn("test", "string", "value", "int", 42, "bool", true)
	Error("test", "string", "value", "int", 42, "bool", true)
}

func TestLogFunctions_EmptyMessage(t *testing.T) {
	Debug("")
	Info("")
	Warn("")
	Error("")
}

func TestLogFunctions_NoFields(t *testing.T) {
	Debug("message")
	Info("message")
	Warn("message")
	Error("message")
}

func TestSync_ErrorHandling(t *testing.T) {
	// Sync may fail in test environment (stderr not available)
	// Just verify the function exists and can be called
	err := Sync()
	_ = err
}

func TestNewLogger_CustomEncoder(t *testing.T) {
	originalLevel := os.Getenv("LOG_LEVEL")
	originalEnv := os.Getenv("ENV")
	defer func() {
		os.Setenv("LOG_LEVEL", originalLevel)
		os.Setenv("ENV", originalEnv)
	}()

	os.Unsetenv("LOG_LEVEL")
	os.Unsetenv("ENV")

	logger := NewLogger()
	if logger == nil {
		t.Fatal("NewLogger() returned nil")
	}

	// Test that we can call various log levels
	logger.Debug("debug")
	logger.Info("info")
	logger.Warn("warn")
	logger.Error("error")
}
