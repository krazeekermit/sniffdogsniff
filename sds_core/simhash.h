#ifndef SIMHASH_H
#define SIMHASH_H

#include "kademlia/kadbucket.h"
#include "common/sdsbytesbuf.h"

#include <iostream>
#include <vector>

#define FNV128_BIT_SIZE 128
#define FNV128_SIZE (FNV128_BIT_SIZE/8)
#define SIMHASH_DIGEST_LENGTH FNV128_BIT_SIZE

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

    friend std::ostream &operator<<(std::ostream &os, const SimHash &id2)
    {
        os << id2.id;
        return os;
    };

private:
    KadId id;

    void init(std::vector<std::string> &tokens);
};

#endif // SIMHASH_H
