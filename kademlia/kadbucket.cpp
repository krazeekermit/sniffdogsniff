#include <string.h>
#include <algorithm>

#include "kadbucket.h"

/*
    KadId
*/

KadId::KadId()
{}

int KadId::height()
{
    int s;
    for (s = KAD_ID_BIT_SZ - 1; s >= 0; s--) {
        unsigned char n = id[s/8];
        if (n != 0 && ((n << (7 - s % 8)) & 0x80) != 0) {
            return s;
        }
    }
    return 0;
}

KadId KadId::operator-(const KadId &id2) const
{
    KadId distance;
    int i;
    for (i = 0; i < KAD_ID_SZ; i++)
        distance.id[i] = id[i] ^ id2.id[i];

    return distance;
}

bool KadId::operator==(const KadId &id2) const
{
    return memcmp(id, id2.id, KAD_ID_SZ) == 0;
}

bool KadId::operator<(const KadId &id2) const
{
    int i;
    for (i = 0; i < KAD_ID_SZ; i++)
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
        memset(newId.id, 0, KAD_ID_SZ);
    }
    return newId;
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
    SHA_CTX ctx;
    SHA1_Init(&ctx);
    SHA1_Update(&ctx, address.c_str(), address.size());
    SHA1_Final(this->id.id, &ctx);
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

bool KadNode::operator<(const KadNode &kn) const
{
    return this->stales < kn.stales;
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

/*
    KadBucket
*/

KadBucket::KadBucket(int height)
    : height(height)
{}

KadBucket::~KadBucket()
{

}

bool KadBucket::pushNode(KadNode &kn)
{
    auto ikn = std::find(this->nodes.begin(), this->nodes.end(), kn);
    bool found = ikn != this->nodes.end();
    if (found) {
        ikn->seenNow();
        ikn->resetStales();
    } else if (this->nodes.size() < KAD_BUCKET_MAX) {
        this->nodes.push_back(kn);
    } else {
        int stalestIdx = -1, staleMax = 0, i;
        for (i = 0; i < this->nodes.size(); i++) {
            KadNode ckn = this->nodes[i];
            if (ckn.getStales() >= STALES_THR && ckn.getStales() > staleMax) {
                staleMax = ckn.getStales();
                stalestIdx = i;
            }
        }

        if (stalestIdx > -1) {
            this->nodes.erase(this->nodes.begin() + stalestIdx);
            this->nodes.push_back(kn);
        } else {
            ikn = std::find(this->replacementNodes.begin(), this->replacementNodes.end(), kn);
            found = ikn != this->replacementNodes.end();
            if (found) {
                ikn->seenNow();
            } else if (this->replacementNodes.size() < KAD_BUCKET_MAX) {
                this->replacementNodes.push_back(kn);
            } else {
                this->replacementNodes.pop_back();
                this->replacementNodes.push_back(kn);
            }
        }
    }

    return found;
}

bool KadBucket::removeNode(KadNode &kn)
{
    auto ikn = std::find(this->nodes.begin(), this->nodes.end(), kn);
    if (ikn != this->nodes.end()) {
        if (this->replacementNodes.size()) {
            this->nodes.erase(std::remove(this->nodes.begin(), this->nodes.end(), kn), this->nodes.end());
            KadNode first = *this->replacementNodes.erase(this->replacementNodes.begin());
            this->nodes.push_back(first);
            reorder();
        } else {
            ikn->incrementStales();
        }
        return true;
    }
    return false;
}

int KadBucket::getHeight() const
{
    return height;
}

//std::vector<KadNode*> KadBucket::getNodes() const
//{
//    return nodes;
//}

//std::vector<KadNode*> KadBucket::getReplacementNodes() const
//{
//    return replacementNodes;
//}

void KadBucket::reorder()
{
    std::sort(this->nodes.begin(), this->nodes.end());
    std::sort(this->replacementNodes.begin(), this->replacementNodes.end());
}

//static KadNode *getNode_(std::vector<KadNode*> &v, KadNode *kn) {
//    for (auto it = v.begin(); it != v.end(); it++) {
//        if ((**it) == (*kn)) {
//            return *it;
//        }
//    }
//    return nullptr;
//}

//KadNode *KadBucket::getNode(KadNode *kn)
//{
//    return getNode_(this->nodes, kn);
//}

//KadNode *KadBucket::getReplacementNode(KadNode *kn)
//{
//    return getNode_(this->replacementNodes, kn);
//}

std::ostream &operator<<(std::ostream &os, const KadBucket &kb2)
{
    os << "KadBucket["
       << "height=" << kb2.height
       << ", nodes=[\n";

    for (auto it = kb2.nodes.begin(); it < kb2.nodes.end(); it++) {
        os << *it << "\n";
    }

    os << "]\n, replacementNodes=[\n";
    for (auto it = kb2.replacementNodes.begin(); it < kb2.replacementNodes.end(); it++) {
        os << *it << "\n";
    }

    os << "]\n]";
    return os;
}
