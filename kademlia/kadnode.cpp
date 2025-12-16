#include "kadnode.h"

#include "net/netutil.h"

/*
    KadId
*/

KadId::KadId()
    : KadId(nullptr)
{}

KadId::KadId(const uint8_t *id_)
{
    if (!id_) {
        return;
    }

    memcpy(this->id, id_, KAD_ID_LENGTH);
}

/*
    Most significant byte is the 16th
    Most significant bit is the first
    in the last non zero byte.
*/
int KadId::height() const
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
    return  this->id[0]  == id2.id[0] &&
            this->id[1]  == id2.id[1] &&
            this->id[2]  == id2.id[2] &&
            this->id[3]  == id2.id[3] &&
            this->id[4]  == id2.id[4] &&
            this->id[5]  == id2.id[5] &&
            this->id[6]  == id2.id[6] &&
            this->id[7]  == id2.id[7] &&
            this->id[8]  == id2.id[8] &&
            this->id[9]  == id2.id[9] &&
            this->id[10] == id2.id[10] &&
            this->id[11] == id2.id[11] &&
            this->id[12] == id2.id[12] &&
            this->id[13] == id2.id[13] &&
            this->id[14] == id2.id[14] &&
            this->id[15] == id2.id[15];
}

bool KadId::operator<(const KadId &id2) const
{
    int i = KAD_ID_LENGTH-1;
    while (i>= 0 && id[i] == id2.id[i])
        i--;

    return id[i] < id2.id[i];
}

std::ostream &operator<<(std::ostream &os, const KadId &id2)
{
    char hexString[33];
    bytes_to_hex_string(hexString, id2.id, KAD_ID_LENGTH);
    os << hexString;

    return os;
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

KadId KadId::fromHexString(const char *hexString)
{
    KadId newId;
    if (hex_string_to_bytes(newId.id, hexString) != 16) {
        throw std::runtime_error("KadId::fromHexString unable to read hex node id string length is not of 32");
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
KadNode::KadNode(KadId id_, std::string address_)
    : id(id_), address(address_), lastSeen(0), stales(0)
{}

KadNode::KadNode()
    : lastSeen(0), stales(0)
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

bool KadNode::operator==(const KadNode &kn) const
{
    return this->id == kn.id;
}

bool KadNode::operator<(const KadNode &kn) const
{
    /* Reverse order */
    return this->lastSeen > kn.lastSeen;
}

const std::string KadNode::getAddress() const
{
    return address;
}

void KadNode::setAddress(const std::string &newAddress)
{
    this->address = newAddress;
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
