#include "kadnode.h"

#include "net/netutil.h"

/*
    KadId
*/

KadId::KadId()
{}

int KadId::height()
{
    int i, d = KAD_ID_BIT_LENGTH - 1;
    for (i = KAD_ID_LENGTH - 1; i >= 0 && id[i] == 0; i--) {
        d -= 8;
    }

    unsigned char n = id[i];
    while ((n & 0x80) == 0) {
        n <<= 1;
        d--;
    }
    return d;
}

KadId KadId::operator-(const KadId &id2) const
{
    KadId distance;

    distance.id[0]  = this->id[0]  ^ id2.id[0];
    distance.id[1]  = this->id[1]  ^ id2.id[1];
    distance.id[2]  = this->id[2]  ^ id2.id[2];
    distance.id[3]  = this->id[3]  ^ id2.id[3];
    distance.id[4]  = this->id[4]  ^ id2.id[4];
    distance.id[5]  = this->id[5]  ^ id2.id[5];
    distance.id[6]  = this->id[6]  ^ id2.id[6];
    distance.id[7]  = this->id[7]  ^ id2.id[7];
    distance.id[8]  = this->id[8]  ^ id2.id[8];
    distance.id[9]  = this->id[9]  ^ id2.id[9];
    distance.id[10] = this->id[10] ^ id2.id[10];
    distance.id[11] = this->id[11] ^ id2.id[11];
    distance.id[12] = this->id[12] ^ id2.id[12];
    distance.id[13] = this->id[13] ^ id2.id[13];
    distance.id[14] = this->id[14] ^ id2.id[14];
    distance.id[15] = this->id[15] ^ id2.id[15];

    return distance;
}

bool KadId::operator==(const KadId &id2) const
{
    return memcmp(id, id2.id, KAD_ID_LENGTH) == 0;
}

bool KadId::operator<(const KadId &id2) const
{
    int i;
    for (i = 0; i < KAD_ID_LENGTH; i++)
        if (id[i] < id2.id[i])
            return true;

    return false;
}

KadId KadId::randomId()
{
    KadId newId;
    FILE *rfp = fopen("/dev/urandom", "rb");
    if (rfp) {
        fread(newId.id, sizeof(newId.id), 1, rfp);
        fclose(rfp);
    } else {
        memset(newId.id, 0, KAD_ID_LENGTH);
    }
    return newId;
}

KadId KadId::idNbitsFarFrom(const KadId &id1, int bdist)
{
    int dIdx = KAD_ID_BIT_LENGTH - bdist;
    KadId ax;
    int i;
    for (i = 0; i < dIdx/8; i++) {
        ax.id[i] = 0x0;
    }

    if (i < KAD_ID_LENGTH) {
        ax.id[i] = 0xff << ((dIdx - i*8) % 8);
    }

    for (i++; i < KAD_ID_LENGTH; i++) {
        ax.id[i] = 0xff;
    }
    return ax - id1;
}

/*
    KadNode
*/

KadNode::KadNode(const char *address_)
    : KadNode(std::string(address_))
{}

KadNode::KadNode(std::string address_)
    : address(address_), lastSeen(0), stales(0)
{
    if (address.size()) {
        char addrBuf[1024];
        address.copy(addrBuf, address.size(), 0);
        char addr[1024];
        /*
            Avoid creation of infinite nodes with the same address but different ports
        */
        net_urlparse(addr, nullptr, nullptr, addrBuf);

        /*
            XOR the two halves of SHA256 hash to obtain an unique 128bit lenght hash,
            this is useful because needs to match with the bit lenght of the FNV1a-128 hash
            that is used to generate the simHash
        */
        unsigned char hash[SHA256_DIGEST_LENGTH];
        SHA256((const unsigned char*) addr, strlen(addr), hash);

        this->id.id[0]  = hash[0]  ^ hash[16];
        this->id.id[1]  = hash[1]  ^ hash[17];
        this->id.id[2]  = hash[2]  ^ hash[18];
        this->id.id[3]  = hash[3]  ^ hash[19];
        this->id.id[4]  = hash[4]  ^ hash[20];
        this->id.id[5]  = hash[5]  ^ hash[21];
        this->id.id[6]  = hash[6]  ^ hash[22];
        this->id.id[7]  = hash[7]  ^ hash[23];
        this->id.id[8]  = hash[8]  ^ hash[24];
        this->id.id[9]  = hash[9]  ^ hash[25];
        this->id.id[10] = hash[10] ^ hash[26];
        this->id.id[11] = hash[11] ^ hash[27];
        this->id.id[12] = hash[12] ^ hash[28];
        this->id.id[13] = hash[13] ^ hash[29];
        this->id.id[14] = hash[14] ^ hash[30];
        this->id.id[15] = hash[15] ^ hash[31];
    }
}

KadNode::KadNode(KadId id_, std::string address_)
    : id(id_), address(address_), lastSeen(0), stales(0)
{}

void KadNode::seenNow()
{
    this->lastSeen = time(nullptr);
}

void KadNode::resetStales()
{
    this->stales = 0;
}

void KadNode::incrementStales()
{
    this->stales++;
}

void KadNode::decrementStales()
{
    this->stales--;
}

bool KadNode::operator==(const KadNode &kn)
{
    return this->id == kn.id;
}

bool KadNode::operator==(const KadNode *kn)
{
    return this->id == kn->id;
}

bool KadNode::operator<(const KadNode &kn) const
{
    /* Reverse order */
    return this->lastSeen > kn.lastSeen;
}

const std::string &KadNode::getAddress() const
{
    return address;
}

time_t KadNode::getLastSeen() const
{
    return lastSeen;
}

int KadNode::getStales() const
{
    return stales;
}

KadId KadNode::getId() const
{
    return this->id;
}

std::ostream &operator<<(std::ostream &os, const KadNode &kn2)
{
    int i;
    os << "KadNode["
       << "id=" << kn2.id
       << ", address=" << kn2.address
       << ", lastSeen=" << kn2.lastSeen
       << ", stales=" << kn2.stales
       << "]";
    return os;
}
