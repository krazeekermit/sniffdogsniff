#ifndef KADNODE_H
#define KADNODE_H

#include "common/macros.h"

#include <openssl/sha.h>
#include <cstdint>
#include <cstring>
#include <ctime>
#include <iostream>

#define KAD_ID_LENGTH      SHA256_DIGEST_LENGTH / 2
#define KAD_ID_BIT_LENGTH  KAD_ID_LENGTH*8

struct KadId
{
    KadId();
    KadId(const uint8_t *id_);

    int height() const;

    KadId operator-(const KadId &id2) const;
    bool operator==(const KadId &id2) const;
    bool operator<(const KadId& id2) const;

    friend std::ostream &operator<<(std::ostream &os, const KadId &id2);

    static KadId randomId();
    static KadId fromHexString(const char *hexString);
    static KadId idNbitsFarFrom(const KadId &id1, int bdist);

    // Member
    unsigned char id[KAD_ID_LENGTH];
};

class KadNode
{
    friend class KadRoutingTable;

public:
    KadNode(KadId id_, std::string address_);

    void seenNow();
    void resetStales();
    void incrementStales();
    void decrementStales();

    bool operator==(const KadNode &kn2) const;
    bool operator<(const KadNode &rhs) const;
    friend std::ostream &operator<<(std::ostream &os, const KadNode &kn2);

    KadId getId() const;

    const std::string getAddress() const;
    void setAddress(const std::string &newAddress);

    time_t getLastSeen() const;
    int getStales() const;

private:
    KadNode();

    KadId id;
    std::string address;
    time_t lastSeen;
    int stales;
};

#endif // KADNODE_H
