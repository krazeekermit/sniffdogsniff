#ifndef LOGGING_HPP
#define LOGGING_HPP

#include <pthread.h>

#include <sstream>
#include <iostream>

#define ANSI_RED     "\e[0;31m"
#define ANSI_CYAN    "\e[0;36m"
#define ANSI_YELLOW  "\e[0;33m"
#define ANSI_WHITE   "\e[0;37m"
#define ANSI_RESET   "\e[0m"

#define USE_COLORS 1

enum LogLevel {
    LOG_LEVEL_INFO, LOG_LEVEL_WARNING, LOG_LEVEL_ERROR, LOG_LEVEL_FATAL, LOG_LEVEL_DEBUG
};

struct Logging {
    friend class Log;
public:
    Logging();

    static void setLevel(LogLevel level);
    static void setLogFile(const char *path);

    pthread_mutex_t mutex;
    LogLevel level;
    FILE *fperr;
    bool logToFile;
};

class Log {
public:
    Log(LogLevel level_);
    ~Log();

    template<typename T>
    Log &operator<<(const T &data)
    {
        this->logStream << data;
        return *this;
    }

private:
    LogLevel level;
    std::ostringstream logStream;
};


#define loginfo Log(LogLevel::LOG_LEVEL_INFO)
#define logerr Log(LogLevel::LOG_LEVEL_ERROR)
#define logwarn Log(LogLevel::LOG_LEVEL_WARNING)
#define logdebug Log(LogLevel::LOG_LEVEL_DEBUG)
#define logfatalerr Log(LogLevel::LOG_LEVEL_FATAL)


#endif // LOGGING_HPP
