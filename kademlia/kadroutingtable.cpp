#include "kadroutingtable.h"

#include "common/macros.h"
#include "common/logging.h"
#include "common/utils.h"

#include <algorithm>

KadRoutingTable::KadRoutingTable()
    : selfNode("")
{
    this->buckets = new KadBucket*[KAD_ID_BIT_LENGTH];

    int i;
    for (i = 0; i < KAD_ID_BIT_LENGTH; i++)
        this->buckets[i] = new KadBucket(i);
}

KadRoutingTable::~KadRoutingTable()
{
    int i;
    for (i = 0; i < KAD_ID_BIT_LENGTH; i++)
        delete this->buckets[i];

    delete this->buckets;
}

KadNode KadRoutingTable::getSelfNode() const
{
    return selfNode;
}

void KadRoutingTable::setSelfNode(KadNode &newSelfNode)
{
    this->selfNode = newSelfNode;
}

bool KadRoutingTable::isFull()
{
    int i;
    for (i = 0; i < KAD_ID_BIT_LENGTH; i++)
        if (!this->buckets[i]->isFull())
            return false;

    return true;
}

bool KadRoutingTable::pushNode(KadNode &kn)
{
    if (this->selfNode == kn)
        return false;

    KadId distance = this->selfNode.getId() - kn.getId();
    return this->buckets[distance.height()]->pushNode(kn);
}

bool KadRoutingTable::removeNode(KadNode &kn)
{
    return this->removeNode(kn.getId());
}

bool KadRoutingTable::removeNode(const KadId &id)
{
    if (this->selfNode.getId() == id)
        return false;

    KadId distance = this->selfNode.getId() - id;
    return this->buckets[distance.height()]->removeNode(id);
}

int KadRoutingTable::getKClosestTo(std::vector<KadNode> &nodes, const KadId &id)
{
    return getClosestTo(nodes, id, KAD_ID_LENGTH);
}

int KadRoutingTable::getClosestTo(std::vector<KadNode> &closest, const KadId &id, int count)
{
    closest.clear();
    if (this->selfNode.getId() == id)
        return 0;

    int height, i, j;
    KadId distance = this->selfNode.getId() - id;
    height = distance.height();

    KadBucket *buck = this->buckets[height];
    for (j = 0; j < buck->nodes.size() && closest.size() < count; j++) {
        closest.push_back(buck->nodes[j]);
    }

    for (i = 1; height - i >= 0 || height + i < KAD_ID_BIT_LENGTH; i++) {
        if (height - i >= 0) {
            buck = this->buckets[height - i];

            for (j = 0; j < buck->nodes.size() && closest.size() < count; j++) {
                closest.push_back(buck->nodes[j]);
            }
        }

        if (height + i < KAD_ID_BIT_LENGTH) {
            buck = this->buckets[height + i];

            for (j = 0; j < buck->nodes.size() && closest.size() < count; j++) {
                closest.push_back(buck->nodes[j]);
            }
        }
    }

    std::sort(closest.begin(), closest.end(), [id](const KadNode &a, const KadNode &b) {
        return (a.getId() - id) < (b.getId() - id);
    });

    return closest.size();
}

const KadNode &KadRoutingTable::getNodeAtHeight(int height, int index)
{
    KadBucket *buck = this->buckets[height];
    return buck->nodes.at(index);
}

int KadRoutingTable::readFile(const char *path)
{
    FILE *fp = fopen(path, "rb");
    if (!fp) {
        logwarn << "kadroutingtable: unable to open cache file " << path;
        return -1;
    }
    int ret, i, j, nodesCount;
    ret = 0;
    for (i = 0; i < KAD_ID_BIT_LENGTH; i++) {
        KadBucket *buck = this->buckets[i] = new KadBucket(i);
        nodesCount = 0;
        if (fread(&nodesCount, sizeof(int32_t), 1, fp) != 1) {
            ret = -1;
            goto end_read;
        }
        for (j = 0; j < nodesCount; j++) {
            KadNode kn("");

            if (fread(kn.id.id, sizeof(unsigned char), KAD_ID_LENGTH, fp) != KAD_ID_LENGTH) {
                ret = -1;
                goto end_read;
            }
            uint16_t addrLen = kn.address.length();
            if (fread(&addrLen, sizeof(uint16_t), 1, fp) != 1) {
                ret = -1;
                goto end_read;
            }
            if (!addrLen) {
                ret = -1;
                goto end_read;
            }
            char buf[512];
            if (fread(buf, sizeof(char), addrLen, fp) != addrLen) {
                ret = -1;
                goto end_read;
            }
            buf[addrLen] = '\0';
            kn.address = std::string(buf);
            if (fread(&kn.lastSeen, sizeof(time_t), 1, fp) != 1) {
                ret = -1;
                goto end_read;
            }
            if (fread(&kn.stales, sizeof(int32_t), 1, fp) != 1) {
                ret = -1;
                goto end_read;
            }

            buck->nodes.push_back(kn);
        }

        if (fread(&nodesCount, sizeof(int32_t), 1, fp) != 1) {
            ret = -1;
            goto end_read;
        }
        for (j = 0; j < nodesCount; j++) {
            KadNode kn("");
            if (fread(kn.id.id, sizeof(unsigned char), KAD_ID_LENGTH, fp) != KAD_ID_LENGTH) {
                ret = -1;
                goto end_read;
            }
            uint16_t addrLen = kn.address.length();
            if (fread(&addrLen, sizeof(uint16_t), 1, fp) != 1) {
                ret = -1;
                goto end_read;
            }
            if (!addrLen) {
                ret = -1;
                goto end_read;
            }
            char buf[512];
            if (fread(buf, sizeof(char), addrLen, fp) != addrLen) {
                ret = -1;
                goto end_read;
            }
            buf[addrLen] = '\0';
            kn.address = std::string(buf);
            if (fread(&kn.lastSeen, sizeof(time_t), 1, fp) != 1) {
                ret = -1;
                goto end_read;
            }
            if (fread(&kn.stales, sizeof(int32_t), 1, fp) != 1) {
                ret = -1;
                goto end_read;
            }

            buck->replacementNodes.push_back(kn);
        }
    }

end_read:
    fclose(fp);

    logdebug << *this;

    return ret;
}

int KadRoutingTable::writeFile(const char *path)
{
    FILE *fp = fopen(path, "wb");
    if (!fp) {
        logwarn << "kadroutingtable: unable to open cache file " << path;
        return -1;
    }
    int ret, i, j, nodesCount;
    ret = 0;
    for (i = 0; i < KAD_ID_BIT_LENGTH; i++) {
        KadBucket *buck = this->buckets[i];
        nodesCount = buck->nodes.size();
        if (fwrite(&nodesCount, sizeof(int32_t), 1, fp) != 1) {
            ret = -1;
            goto end_write;
        }
        for (j = 0; j < nodesCount; j++) {
            KadNode kn = buck->nodes[j];
            if (fwrite(kn.id.id, sizeof(unsigned char), KAD_ID_LENGTH, fp) != KAD_ID_LENGTH) {
                ret = -1;
                goto end_write;
            }
            uint16_t addrLen = kn.address.length();
            if (fwrite(&addrLen, sizeof(uint16_t), 1, fp) != 1) {
                ret = -1;
                goto end_write;
            }
            if (fwrite(kn.address.c_str(), sizeof(char), kn.address.length(), fp) != kn.address.length()) {
                ret = -1;
                goto end_write;
            }
            if (fwrite(&kn.lastSeen, sizeof(time_t), 1, fp) != 1) {
                ret = -1;
                goto end_write;
            }
            if (fwrite(&kn.stales, sizeof(int32_t), 1, fp) != 1) {
                ret = -1;
                goto end_write;
            }
        }

        nodesCount = buck->replacementNodes.size();
        if (fwrite(&nodesCount, sizeof(int32_t), 1, fp) != 1) {
            ret = -1;
            goto end_write;
        }
        for (j = 0; j < nodesCount; j++) {
            KadNode kn = buck->replacementNodes[j];
            if (fwrite(kn.id.id, sizeof(unsigned char), KAD_ID_LENGTH, fp) != KAD_ID_LENGTH) {
                ret = -1;
                goto end_write;
            }
            uint16_t addrLen = kn.address.length();
            if (fwrite(&addrLen, sizeof(uint16_t), 1, fp) != 1) {
                ret = -1;
                goto end_write;
            }
            if (fwrite(kn.address.c_str(), sizeof(char), kn.address.length(), fp) != kn.address.length()) {
                ret = -1;
                goto end_write;
            }
            if (fwrite(&kn.lastSeen, sizeof(time_t), 1, fp) != 1) {
                ret = -1;
                goto end_write;
            }
            if (fwrite(&kn.stales, sizeof(int32_t), 1, fp) != 1) {
                ret = -1;
                goto end_write;
            }
        }
    }

end_write:
    fclose(fp);

    if (ret)
        logdebug << "error in writing ktable";
    return ret;
}

std::ostream &operator<<(std::ostream &os, const KadRoutingTable &kt2)
{
    os << "KadRoutingTable["
       << ", buckets=[\n";

    int i;
    for (i = 0; i < KAD_ID_BIT_LENGTH; i++) {
        os << *kt2.buckets[i] << "\n";
    }

    os << "]";
    return os;
}

std::vector<KadNode> KadRoutingTable::getAllNodes()
{
    int i;
    std::vector<KadNode> allNodes;
    for (i = 0; i < KAD_ID_BIT_LENGTH; i++) {
        std::vector<KadNode> on = this->buckets[i]->nodes;
        allNodes.insert(allNodes.end(), on.begin(), on.end());
    }
    return allNodes;
}
