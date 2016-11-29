package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	DeepCall    = 4
	LevelSilent = iota
	LevelDebug
	LevelInfo
	LevelWarn
	LevelError
	LevelTerminate
)

var level int

var DebugLevel bool

func logLevel() int {
	levelEnv := strings.ToUpper(os.Getenv("LOGLEVEL"))
	switch levelEnv {
	case "DEBUG":
		{
			log.Println("Logger debug level on")
			return LevelDebug
		}
	case "INFO":
		return LevelInfo
	case "WARN":
		return LevelWarn
	case "ERROR":
		return LevelError
	case "SILENT":
		return LevelSilent
	default:
		return LevelInfo
	}
}

func getPath() string {
	result := ""
	if pc, file, line, ok := runtime.Caller(DeepCall); ok {
		name := filepath.Base(runtime.FuncForPC(pc).Name())
		result = fmt.Sprintf("%s %s:%d", name, file, line)
	}
	return result
}

func getLevelName(logLevel int) string {
	switch logLevel {
	case LevelSilent:
		return "SILENT"
	case LevelInfo:
		return "INFO"
	case LevelDebug:
		return "DEBUG"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	case LevelTerminate:
		return "FATAL"
	default:
		return "INFO"
	}
}

func outLog(logLevel int, msg *string) {
	stackInfo := ""
	if logLevel != LevelInfo && logLevel != LevelWarn {
		stackInfo = getPath()
	}
	if logLevel == LevelTerminate {
		panic(fmt.Sprintf("%s %s", stackInfo, *msg))
	} else {
		log.Printf(
			"%s-> %s %s\n", getLevelName(logLevel), stackInfo, *msg)
	}
}

func out(logLevel int, format string, a ...interface{}) {
	if logLevel >= level {
		msg := fmt.Sprintf(format, a...)
		outLog(logLevel, &msg)
	}
}

func Info(format string, a ...interface{}) {
	out(LevelInfo, format, a...)
}

func Debug(format string, a ...interface{}) {
	out(LevelDebug, format, a...)
}

func Warn(format string, a ...interface{}) {
	out(LevelWarn, format, a...)
}

func Error(format string, a ...interface{}) {
	out(LevelError, format, a...)
}

func Terminate(format string, a ...interface{}) {
	out(LevelTerminate, format, a...)
}

func init() {
	level = logLevel()
	DebugLevel = level == LevelDebug
}
