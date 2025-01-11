#ifndef SIMHASH_H
#define SIMHASH_H

#include "kademlia/kadbucket.h"

#include <iostream>
#include <vector>

class SimHash {
public:
    SimHash() = default;
    ~SimHash() = default;

    void update(std::string str);
    KadId digest() const;

    static const KadId digest(std::string str);

private:
    std::vector<std::string> tokens;
};

#endif // SIMHASH_H
