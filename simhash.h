#ifndef SIMHASH_H
#define SIMHASH_H

#include "kademlia/kadbucket.h"
#include "sdsbytesbuf.h"

#include <iostream>
#include <vector>

class SimHash {
public:
    SimHash() = default;
    SimHash(std::string str);
    SimHash(std::vector<std::string> &tokens);
    ~SimHash() = default;

    KadId getId() const;
    int distance(SimHash &other);

    void read(SdsBytesBuf &buf);
    void write(SdsBytesBuf &buf);

private:
    KadId id;

    void init(std::vector<std::string> &tokens);
};

#endif // SIMHASH_H
