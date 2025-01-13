#include "logging.h"

static Logging _logging;

Logging::Logging()
    : level(LogLevel::LOG_LEVEL_DEBUG), fperr(stderr)
{
    pthread_mutex_init(&this->mutex, nullptr);
}

void Logging::setLevel(LogLevel level)
{
    _logging.level = level;
}

void Logging::setLogFile(const char *path)
{

}

Log::Log(LogLevel level_)
    : level(level_)
{}

Log::~Log()
{
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

    pthread_mutex_lock(&_logging.mutex);
#if USE_COLORS
    switch (level) {
    case LOG_LEVEL_WARNING:
        fprintf(_logging.fperr, ANSI_YELLOW);
        break;
    case LOG_LEVEL_FATAL:
    case LOG_LEVEL_ERROR:
        fprintf(_logging.fperr, ANSI_RED);
        break;
    case LOG_LEVEL_DEBUG:
        fprintf(_logging.fperr, ANSI_CYAN);
        break;
    default:
        break;
    }
#endif
    fprintf(_logging.fperr, "[%d/%02d/%02d %02d:%02d:%02d] [%5s] %s", tm.tm_year + 1900, tm.tm_mon + 1, tm.tm_mday, tm.tm_hour, tm.tm_min, tm.tm_sec, slevel, this->logStream.str().c_str());
#if USE_COLORS
    fprintf(_logging.fperr, ANSI_RESET);
#endif
    fprintf(_logging.fperr, "\n");
    pthread_mutex_unlock(&_logging.mutex);

    if (this->level == LOG_LEVEL_FATAL)
        exit(-1);
}
