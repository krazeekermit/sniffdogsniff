package logging

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	INFO  int = 0
	WARN  int = 1
	ERROR int = 2
	TRACE int = 3
)

const (
	INFO_STR  = "info"
	WARN_STR  = "warn"
	ERROR_STR = "error"
	TRACE_STR = "trace"
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
var mutex *sync.Mutex
var outStream *bufio.Writer

func StrToLogLevel(levelStr string) int {
	switch strings.ToLower(levelStr) {
	case INFO_STR:
		return INFO
	case WARN_STR:
		return WARN
	case ERROR_STR:
		return ERROR
	case TRACE_STR:
		return TRACE
	}
	return TRACE
}

func SprintTrimmed(a ...any) string {
	return strings.Trim(fmt.Sprintln(a...), "\n")
}

func InitLogging(level int) {
	logLevel = level
}

func printlog(level, color, message string) {
	if mutex == nil {
		mutex = &sync.Mutex{}
	}
	if outStream == nil {
		outStream = bufio.NewWriter(os.Stdout)
	}

	end := ANSI_END
	if noAnsi {
		color = ""
		end = ""
	}

	mutex.Lock()
	fmt.Fprintf(outStream, "[%s %s%5s%s] %s\n", time.Now().Local().Format(time.ANSIC), color, level, end, message)
	outStream.Flush()
	mutex.Unlock()
}

func SetLoggingToFile(path string) {
	fp, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		panic("failed to create log file")
	}

	noAnsi = true
	outStream = bufio.NewWriter(fp)
}

func LogInfo(a ...any) {
	if logLevel >= INFO {
		printlog(INFO_STR, ANSI_END, SprintTrimmed(a...))
	}
}

func LogWarn(a ...any) {
	if logLevel >= WARN {
		printlog(WARN_STR, ANSI_YELLOW, SprintTrimmed(a...))
	}
}

func LogError(a ...any) {
	if logLevel >= ERROR {
		printlog(ERROR_STR, ANSI_RED, SprintTrimmed(a...))
	}
}

func LogTrace(a ...any) {
	if logLevel >= TRACE {
		printlog(TRACE_STR, ANSI_CYAN, SprintTrimmed(a...))
	}
}
