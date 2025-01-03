#ifndef LOGGING_HPP
#define LOGGING_HPP

#include <sstream>
#include <iostream>

#define ANSI_RED     "\e[0;31m"
#define ANSI_GREEN   "\e[0;32m"
#define ANSI_YELLOW  "\e[0;33m"
#define ANSI_WHITE   "\e[0;37m"
#define ANSI_RESET   "\e[0m"

#define USE_COLORS 1

enum LogLevel {
    LOG_LEVEL_INFO, LOG_LEVEL_WARNING, LOG_LEVEL_ERROR, LOG_LEVEL_DEBUG
};

struct Logging {
    friend class Log;
public:
    Logging()
        : level(LogLevel::LOG_LEVEL_DEBUG), fperr(std::cerr), fpout(std::cout)
    {}

    LogLevel level;
    std::ostream &fperr;
    std::ostream &fpout;
};

static Logging _logging;

class Log {
public:

    Log(LogLevel level_)
        : level(level_)
    {
#if USE_COLORS
        switch (level) {
        case LOG_LEVEL_ERROR:
            _logging.fperr << ANSI_RED;
            break;
        case LOG_LEVEL_DEBUG:
            _logging.fperr << ANSI_GREEN;
            break;
        default:
            break;
        }
#endif

        time_t t = time(NULL);
        struct tm tm = *localtime(&t);
        _logging.fperr << tm.tm_year + 1900 << "/" <<tm.tm_mon + 1 << "/" << tm.tm_mday << " " << tm.tm_hour << ":" << tm.tm_min << ":"  << tm.tm_sec;
        switch (level) {
        case LOG_LEVEL_INFO:
            _logging.fperr << " info ";
            break;
        case LOG_LEVEL_ERROR:
            _logging.fperr << " error ";
            break;
        case LOG_LEVEL_DEBUG:
            _logging.fperr << " debug ";
            break;
        default:
            break;
        }
    }

    ~Log()
    {
#if USE_COLORS
        _logging.fperr << ANSI_RESET;
#endif
    }

    template<typename T>
    Log& operator<< (const T& data)
    {
        _logging.fperr << data;
        return *this;
    }

private:
    LogLevel level;
};


#define loginfo Log(LogLevel::LOG_LEVEL_INFO)
#define logerr Log(LogLevel::LOG_LEVEL_ERROR)
#define logdebug Log(LogLevel::LOG_LEVEL_DEBUG)
#define logfatalerr Log(LogLevel::LOG_LEVEL_DEBUG)


#endif // LOGGING_HPP
