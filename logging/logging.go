package logging

import (
	"fmt"
	"log"
	"os"
	"strings"
)

const (
	INFO  int = 0
	WARN  int = 1
	ERROR int = 2
	DEBUG int = 3
)

const (
	INFO_STR  = "info"
	WARN_STR  = "warn"
	ERROR_STR = "error"
	DEBUG_STR = "debug"
)

const (
	ANSI_RED    string = "\033[31m"
	ANSI_YELLOW string = "\033[33m"
	ANSI_WHITE  string = "\033[97m"
	ANSI_CYAN   string = "\033[36m"
	ANSI_END    string = "\033[0m"
)

var logLevel int
var noAnsi bool = false

func StrToLogLevel(levelStr string) int {
	switch strings.ToLower(levelStr) {
	case INFO_STR:
		return INFO
	case WARN_STR:
		return WARN
	case ERROR_STR:
		return ERROR
	case DEBUG_STR:
		return DEBUG
	}
	return DEBUG
}

func SprintTrimmed(a ...any) string {
	return strings.Trim(fmt.Sprintln(a...), "\n")
}

func InitLogging(level int) {
	log.SetFlags(log.Ldate | log.Ltime)
	logLevel = level
}

func printlog(level, color, sender, message string) {
	end := ANSI_END
	if noAnsi {
		color = ""
		end = ""
	}

	log.Printf("%s%5s%s [%s] %s\n", color, level, end, sender, message)

}

func SetLoggingToFile(path string) {
	fp, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		panic("failed to create log file")
	}

	log.SetOutput(fp)
	noAnsi = true
}

func Infof(sender, format string, a ...any) {
	if logLevel >= INFO {
		printlog(INFO_STR, ANSI_END, sender, fmt.Sprintf(format, a...))
	}
}

func Warnf(sender, format string, a ...any) {
	if logLevel >= WARN {
		printlog(WARN_STR, ANSI_YELLOW, sender, fmt.Sprintf(format, a...))
	}
}

func Errorf(sender, format string, a ...any) {
	if logLevel >= ERROR {
		printlog(ERROR_STR, ANSI_RED, sender, fmt.Sprintf(format, a...))
	}
}

func Debugf(sender, format string, a ...any) {
	if logLevel >= DEBUG {
		printlog(DEBUG_STR, ANSI_CYAN, sender, fmt.Sprintf(format, a...))
	}
}
