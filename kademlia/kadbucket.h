#ifndef KADBUCKET_H
#define KADBUCKET_H

#include "common/macros.h"

#include <openssl/sha.h>
#include <cstdint>
#include <cstring>
#include <ctime>
#include <iostream>
#include <vector>

#define KAD_ID_LENGTH      SHA256_DIGEST_LENGTH / 2
#define KAD_ID_BIT_LENGTH  KAD_ID_LENGTH*8
#define KAD_BUCKET_MAX 20
#define STALES_THR     5

struct KadId
{
    KadId();

    int height();

    KadId operator-(const KadId &id2) const;
    bool operator==(const KadId &id2) const;
    bool operator<(const KadId& id2) const;

    friend std::ostream &operator<<(std::ostream &os, const KadId &id2)
    {
        STREAM_HEX(os, id2.id, KAD_ID_LENGTH);
        return os;
    };

    static KadId randomId();
    static KadId idNbitsFarFrom(const KadId &id1, int bdist);

    // Member
    unsigned char id[KAD_ID_LENGTH];
};

class KadNode
{
    friend class KadRoutingTable;

public:
    KadNode(const char *address);
    KadNode(std::string address);
    KadNode(KadId id_, std::string address_);

    void seenNow();
    void resetStales();
    void incrementStales();
    void decrementStales();

    bool operator==(const KadNode &kn2);
    bool operator<(const KadNode &rhs) const;
    friend std::ostream &operator<<(std::ostream &os, const KadNode &kn2);

    KadId getId() const;
    const std::string &getAddress() const;
    time_t getLastSeen() const;
    int getStales() const;

private:
    KadId id;
    std::string address;
    time_t lastSeen;
    int stales;
};


class KadBucket
{
    friend class KadRoutingTable;

public:
    KadBucket(int height);
    ~KadBucket();

    bool pushNode(KadNode &kn);
    bool removeNode(KadNode &kn);
    bool removeNode(const KadId &id);

    friend std::ostream &operator<<(std::ostream &os, const KadBucket &kb2);

    int getHeight() const;
    bool isFull();

private:
    int height;
    std::vector<KadNode> nodes;
    std::vector<KadNode> replacementNodes;

    void reorder();
//    getNode(KadNode *kn);
//    KadNode* getReplacementNode(KadNode *kn);
};

#endif // KADBUCKET_H
