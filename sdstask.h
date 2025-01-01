#ifndef SDSTIMER_H
#define SDSTIMER_H

#include <pthread.h>
#include <time.h>

#include <functional>

class SdsTask
{
public:
    SdsTask(std::function<void()> task_, time_t delay_, bool detach_ = true);
    ~SdsTask();

    void stop();

protected:
    static void *threadTask(void *p);

private:
    pthread_t tthread;
    bool detach;
    int run;
    time_t delay;
    std::function<void()> task;

};

#endif // SDSTIMER_H
