package logger

import (
	"fmt"
	"log"
	"os"
	"strings"
)

// LogLevel represents the logging level
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

var (
	currentLevel = INFO // Default to INFO
	levelNames   = map[LogLevel]string{
		DEBUG: "DEBUG",
		INFO:  "INFO",
		WARN:  "WARN",
		ERROR: "ERROR",
	}
	levelEmojis = map[LogLevel]string{
		DEBUG: "🔍",
		INFO:  "ℹ️ ",
		WARN:  "⚠️ ",
		ERROR: "❌",
	}
)

// SetLevel sets the global logging level
func SetLevel(level LogLevel) {
	currentLevel = level
}

// SetLevelFromString sets the logging level from a string (DEBUG, INFO, WARN, ERROR)
func SetLevelFromString(levelStr string) {
	switch strings.ToUpper(levelStr) {
	case "DEBUG":
		currentLevel = DEBUG
	case "INFO":
		currentLevel = INFO
	case "WARN", "WARNING":
		currentLevel = WARN
	case "ERROR":
		currentLevel = ERROR
	default:
		log.Printf("Unknown log level: %s, defaulting to INFO", levelStr)
		currentLevel = INFO
	}
}

// Debug logs a debug message
func Debug(format string, v ...interface{}) {
	logMessage(DEBUG, format, v...)
}

// Info logs an info message
func Info(format string, v ...interface{}) {
	logMessage(INFO, format, v...)
}

// Warn logs a warning message
func Warn(format string, v ...interface{}) {
	logMessage(WARN, format, v...)
}

// Error logs an error message
func Error(format string, v ...interface{}) {
	logMessage(ERROR, format, v...)
}

// logMessage logs a message if the level is enabled
func logMessage(level LogLevel, format string, v ...interface{}) {
	if level < currentLevel {
		return
	}

	emoji := levelEmojis[level]
	levelName := levelNames[level]
	message := fmt.Sprintf(format, v...)

	log.Printf("%s [%s] %s", emoji, levelName, message)
}

// InitFromEnv initializes the logger from environment variable
func InitFromEnv() {
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel != "" {
		SetLevelFromString(logLevel)
		Info("Log level set to: %s", strings.ToUpper(logLevel))
	}
}
