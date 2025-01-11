#include "logging.h"

static Logging _logging;

Logging::Logging()
    : level(LogLevel::LOG_LEVEL_DEBUG), fperr(std::cerr), fpout(std::cout)
{}

void Logging::setLevel(LogLevel level)
{
    _logging.level = level;
}

void Logging::setLogFile(const char *path)
{

}

Log::Log(LogLevel level_)
    : level(level_)
{
#if USE_COLORS
    _logging.fperr << ANSI_RESET;
    switch (level) {
    case LOG_LEVEL_WARNING:
        _logging.fperr << ANSI_YELLOW;
        break;
    case LOG_LEVEL_FATAL:
    case LOG_LEVEL_ERROR:
        _logging.fperr << ANSI_RED;
        break;
    case LOG_LEVEL_DEBUG:
        _logging.fperr << ANSI_CYAN;
        break;
    default:
        break;
    }
#endif
    const char *slevel = "";

    switch (level) {
    case LOG_LEVEL_INFO:
        slevel = "info";
        break;
    case LOG_LEVEL_WARNING:
        slevel = "warn";
        break;
    case LOG_LEVEL_FATAL:
        slevel = "fatal";
        break;
    case LOG_LEVEL_ERROR:
        slevel = "error";
        break;
    case LOG_LEVEL_DEBUG:
        slevel = "debug";
        break;
    default:
        break;
    }

    time_t t = time(NULL);
    struct tm tm = *localtime(&t);
    char fmt[1024];
    sprintf(fmt, "\n[%d/%02d/%02d %02d:%02d:%02d] [%5s] ", tm.tm_year + 1900, tm.tm_mon + 1, tm.tm_mday, tm.tm_hour, tm.tm_min, tm.tm_sec, slevel);

    _logging.fperr << fmt;
}

Log::~Log()
{
    if (this->level == LOG_LEVEL_FATAL)
        exit(-1);
}

std::ostream &Log::logStream()
{
    switch (level) {
    case LOG_LEVEL_INFO:
    case LOG_LEVEL_WARNING:
        return _logging.fpout;
    case LOG_LEVEL_FATAL:
    case LOG_LEVEL_ERROR:
    case LOG_LEVEL_DEBUG:
    default:
        return _logging.fperr;
    }
}
