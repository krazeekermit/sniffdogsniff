#include "sdstask.h"

#include "common/logging.h"

SdsTask::SdsTask(time_t delay_, bool detach_)
    : delay(delay_), running(false), detach(detach_)
{
}

SdsTask::~SdsTask()
{
    this->stop();
}

bool SdsTask::isRunning()
{
    return this->running;
}

void SdsTask::start()
{
    if (!this->running) {
        this->running = true;
        if (this->detach) {
            pthread_create(&this->tthread, nullptr, &SdsTask::executeFunc, this);
        } else {
            SdsTask::executeFunc(this);
        }
    }
}

void SdsTask::stop()
{
    this->running = false;
    if (this->detach) {
        pthread_join(this->tthread, nullptr);
    }
}

void *SdsTask::executeFunc(void *p)
{
    SdsTask *task = static_cast<SdsTask*>(p);
    time_t msec;

    while (task->running) {
        task->execute();

        clock_t startTime = clock();
        do {
          clock_t delta = clock() - startTime;
          msec = delta * 1000 / CLOCKS_PER_SEC;
        } while (task->running && msec <  task->delay);
    }

    return nullptr;
}
