#include "sdstask.h"

SdsTask::SdsTask(std::function<void ()> task_, time_t delay_, bool detach_)
    : task(task_), delay(delay_), run(1), detach(detach_)
{
    if (this->detach)
        pthread_create(&this->tthread, nullptr, &SdsTask::threadTask, this);
    else
        SdsTask::threadTask(this);
}

SdsTask::~SdsTask()
{
    if (this->detach) {
        pthread_join(this->tthread, nullptr);
    }
}

void SdsTask::stop()
{
    this->run = 0;
}

void *SdsTask::threadTask(void *p)
{
    SdsTask *timer = static_cast<SdsTask*>(p);
    clock_t startTime = clock();
    time_t msec;

    while (timer->run) {
        timer->task();

        do {
          clock_t delta = clock() - startTime;
          msec = delta * 1000 / CLOCKS_PER_SEC;
        } while ( msec <  timer->delay);
    }

    return nullptr;
}
