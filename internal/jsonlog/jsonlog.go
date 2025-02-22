package jsonlog

import (
	"encoding/json"
	"io"
	"os"
	"runtime/debug"
	"sync"
	"time"
)

type Level int

const (
	LevelInfo Level = iota
	LevelError
	LevelFatal
	LevelOff
)

func (level Level) string() string {
	switch level {
	case LevelInfo:
		return "INFO"
	case LevelError:
		return "Error"
	case LevelFatal:
		return "FATAL"
	default:
		return ""
	}
}

type Logger struct {
	out      io.Writer
	minLevel Level
	mutex    sync.Mutex
}

func New(out io.Writer, minLevel Level) *Logger {
	return &Logger{
		out:      out,
		minLevel: minLevel,
	}
}

func (logger *Logger) Print(level Level, message string, properties map[string]string) (int, error) {
	if level < logger.minLevel {
		return 0, nil
	}

	aux := struct {
		Level      string
		Time       string
		Message    string
		Properties map[string]string
		Trace      string
	}{
		Level:      level.string(),
		Time:       time.Now().UTC().Format(time.RFC3339),
		Message:    message,
		Properties: properties,
	}

	if level >= LevelError {
		aux.Trace = string(debug.Stack())
	}

	var line []byte
	line, err := json.Marshal(aux)
	if err != nil {
		line = []byte(LevelError.string() + "unable to marshal log message" + err.Error())
	}

	logger.mutex.Lock()
	defer logger.mutex.Unlock()

	return logger.out.Write(append(line, '\n'))
}

func (Logger *Logger) PrintInfo(message string, properties map[string]string) {
	Logger.Print(LevelInfo, message, properties)
}

func (Logger *Logger) PrintError(err error, properties map[string]string) {
	Logger.Print(LevelError, err.Error(), properties)
}

func (logger *Logger) PrintFatal(err error, properties map[string]string) {
	logger.Print(LevelFatal, err.Error(), properties)
	os.Exit(1)
}

// just to align with io.writer interface so you can use it inside the http server as a parameter
func (logger *Logger) Write(message []byte) (int, error) {
	return logger.Print(LevelError, string(message), nil)
}
