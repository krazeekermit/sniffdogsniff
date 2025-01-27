#ifndef SDSTIMER_H
#define SDSTIMER_H

#include <pthread.h>
#include <time.h>

#include <functional>

class SdsTask
{
public:
    SdsTask(time_t delay_, bool detach_ = true);
    virtual ~SdsTask();

    bool isRunning();
    void start();
    void stop();

protected:
    virtual int execute() = 0;

private:
    static void *executeFunc(void *p);

    pthread_t tthread;
    bool detach;
    bool running;
    time_t delay;
};

#endif // SDSTIMER_H
