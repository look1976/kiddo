package logger

import (
	"os"
	"path/filepath"

	"github.com/natefinch/lumberjack"
	"github.com/sirupsen/logrus"
)

var logger *logrus.Logger

func init() {
	logger = logrus.New()
}

// Init initializes the logger with file output and rotation
func Init() error {
	// Ensure log directory exists
	logDir := filepath.Join(os.Getenv("ProgramData"), "Kiddo", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	logPath := filepath.Join(logDir, "kiddo.log")

	// Set up lumberjack for log rotation (7 days, daily rotation)
	logFile := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    10, // megabytes
		MaxBackups: 7,  // days of logs to keep
		MaxAge:     7,  // days
		Compress:   true,
	}

	// Multi-writer: write to file and stdout (when running in console)
	logger.SetOutput(logFile)
	logger.SetLevel(logrus.DebugLevel)

	// Use JSON formatter for structured logging
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
	})

	logger.Infof("Logger initialized, logs will be written to: %s", logPath)
	return nil
}

// Get returns the global logger instance
func Get() *logrus.Logger {
	return logger
}
