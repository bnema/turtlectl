package logger

import (
	"io"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
)

func init() {
	// Silence the default charmbracelet/log logger
	// All logging should go through our custom logger instance
	log.SetLevel(log.FatalLevel)
}

var (
	// Log is the global logger instance
	Log *log.Logger

	// logFile is the file handle for the log file
	logFile *os.File
)

// Init initializes the logger with the given verbosity level
// When verbose is false, logs go to file only
// When verbose is true, logs go to both file and stderr
func Init(verbose bool) error {
	// Get log file path
	cacheDir := os.Getenv("XDG_CACHE_HOME")
	if cacheDir == "" {
		homeDir, _ := os.UserHomeDir()
		cacheDir = filepath.Join(homeDir, ".cache")
	}
	logDir := filepath.Join(cacheDir, "turtle-wow")
	logPath := filepath.Join(logDir, "turtlectl.log")

	// Ensure log directory exists
	if err := os.MkdirAll(logDir, 0755); err != nil {
		// Fall back to stderr only if we can't create log dir
		Log = log.NewWithOptions(os.Stderr, log.Options{
			ReportTimestamp: true,
		})
		if verbose {
			Log.SetLevel(log.DebugLevel)
		} else {
			Log.SetLevel(log.WarnLevel)
		}
		return nil
	}

	// Open log file (append mode)
	var err error
	logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		// Fall back to stderr only
		Log = log.NewWithOptions(os.Stderr, log.Options{
			ReportTimestamp: true,
		})
		if verbose {
			Log.SetLevel(log.DebugLevel)
		} else {
			Log.SetLevel(log.WarnLevel)
		}
		return nil
	}

	// Set up output destination
	var output io.Writer
	if verbose {
		// Verbose: write to both file and stderr
		output = io.MultiWriter(logFile, os.Stderr)
	} else {
		// Normal: write to file only
		output = logFile
	}

	Log = log.NewWithOptions(output, log.Options{
		ReportTimestamp: true,
	})

	if verbose {
		Log.SetLevel(log.DebugLevel)
	} else {
		Log.SetLevel(log.InfoLevel)
	}

	return nil
}

// Close closes the log file
func Close() {
	if logFile != nil {
		_ = logFile.Close()
	}
}

// GetLogPath returns the path to the log file
func GetLogPath() string {
	cacheDir := os.Getenv("XDG_CACHE_HOME")
	if cacheDir == "" {
		homeDir, _ := os.UserHomeDir()
		cacheDir = filepath.Join(homeDir, ".cache")
	}
	return filepath.Join(cacheDir, "turtle-wow", "turtlectl.log")
}

// Convenience functions that use the global logger

func Debug(msg interface{}, keyvals ...interface{}) {
	if Log != nil {
		Log.Debug(msg, keyvals...)
	}
}

func Info(msg interface{}, keyvals ...interface{}) {
	if Log != nil {
		Log.Info(msg, keyvals...)
	}
}

func Warn(msg interface{}, keyvals ...interface{}) {
	if Log != nil {
		Log.Warn(msg, keyvals...)
	}
}

func Error(msg interface{}, keyvals ...interface{}) {
	if Log != nil {
		Log.Error(msg, keyvals...)
	}
}

func Fatal(msg interface{}, keyvals ...interface{}) {
	if Log != nil {
		Log.Fatal(msg, keyvals...)
	}
}
