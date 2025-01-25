#ifndef SDSTIMER_H
#define SDSTIMER_H

#include <pthread.h>
#include <time.h>

#include <functional>

class SdsTask
{
public:
    SdsTask(std::function<void(SdsTask *timer)> task_, time_t delay_, bool detach_ = true);
    ~SdsTask();

    bool isRunning();

    void stop();

protected:
    static void *runTask(void *p);

private:
    pthread_t tthread;
    bool detach;
    bool run;
    time_t delay;
    std::function<void(SdsTask *timer)> task;

};

#endif // SDSTIMER_H
