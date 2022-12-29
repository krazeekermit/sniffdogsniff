package logging

import (
	"fmt"
	"log"
	"strings"
)

const (
	INFO  int = 0
	WARN  int = 1
	ERROR int = 2
	TRACE int = 3
)

const (
	ANSI_RED    string = "\033[31m"
	ANSI_YELLOW string = "\033[33m"
	ANSI_WHITE  string = "\033[97m"
	ANSI_CYAN   string = "\033[36m"
	ANSI_END    string = "\033[0m"
)

var logLevel int

func SprintTrimmed(a ...any) string {
	return strings.Trim(fmt.Sprintln(a...), "\n")
}

func InitLogging(level int) {
	logLevel = level
}

func LogInfo(a ...any) {
	if logLevel >= INFO {
		log.Println(ANSI_WHITE, "[INFO]", SprintTrimmed(a...), ANSI_END)
	}
}

func LogWarn(a ...any) {
	if logLevel >= WARN {
		log.Println(ANSI_YELLOW, "[WARN]", SprintTrimmed(a...), ANSI_END)
	}
}

func LogError(a ...any) {
	if logLevel >= ERROR {
		log.Println(ANSI_RED, "[ERROR]", SprintTrimmed(a...), ANSI_END)
	}
}

func LogTrace(a ...any) {
	if logLevel >= TRACE {
		log.Println(ANSI_CYAN, "[TRACE]", SprintTrimmed(a...), ANSI_END)
	}
}
