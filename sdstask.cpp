#include "sdstask.h"

SdsTask::SdsTask(std::function<void (SdsTask *timer)> task_, time_t delay_, bool detach_)
    : task(task_), delay(delay_), run(true), detach(detach_)
{
    if (this->detach)
        pthread_create(&this->tthread, nullptr, &SdsTask::runTask, this);
    else
        SdsTask::runTask(this);
}

SdsTask::~SdsTask()
{
    this->stop();
}

bool SdsTask::isRunning()
{
    return this->run;
}

void SdsTask::stop()
{
    this->run = false;
    if (this->detach) {
        pthread_join(this->tthread, nullptr);
    }
}

void *SdsTask::runTask(void *p)
{
    SdsTask *timer = static_cast<SdsTask*>(p);
    time_t msec;

    while (timer->run) {
        timer->task(timer);

        clock_t startTime = clock();
        do {
          clock_t delta = clock() - startTime;
          msec = delta * 1000 / CLOCKS_PER_SEC;
        } while (timer->run && msec <  timer->delay);
    }

    return nullptr;
}
