#include "kadroutingtable.h"

#include "common/macros.h"
#include "common/logging.h"
#include "common/utils.h"

#include <algorithm>

KadRoutingTable::KadRoutingTable()
    : selfNode("")
{
    this->buckets = new KadBucket*[KAD_ID_BIT_SZ];

    int i;
    for (i = 0; i < KAD_ID_BIT_SZ; i++)
        this->buckets[i] = new KadBucket(i);
}

KadRoutingTable::~KadRoutingTable()
{
    int i;
    for (i = 0; i < KAD_ID_BIT_SZ; i++)
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
    for (i = 0; i < KAD_ID_BIT_SZ; i++)
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
    return getClosestTo(nodes, id, KAD_ID_SZ);
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

    for (i = 1; height - i >= 0 || height + i < KAD_ID_BIT_SZ; i++) {
        if (height - i >= 0) {
            buck = this->buckets[height - i];

            for (j = 0; j < buck->nodes.size() && closest.size() < count; j++) {
                closest.push_back(buck->nodes[j]);
            }
        }

        if (height + i < KAD_ID_BIT_SZ) {
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
    for (i = 0; i < KAD_ID_BIT_SZ; i++) {
        KadBucket *buck = this->buckets[i] = new KadBucket(i);
        // GOTO_IF(fwrite(&buck->height, sizeof(int32_t), 1, fp) != sizeof(int32_t), end_write, ret, -1);
        nodesCount = 0;
        GOTO_IF(fread(&nodesCount, sizeof(int32_t), 1, fp) != sizeof(int32_t), end_read, ret, -1);
        for (j = 0; j < nodesCount; j++) {
            KadNode kn("");
            GOTO_IF(fread(kn.id.id, sizeof(unsigned char), KAD_ID_SZ, fp) != KAD_ID_SZ, end_read, ret, -1);
            GOTO_IF(fgetstdstr(kn.address, fp) == 0, end_read, ret, -1);
            GOTO_IF(fread(&kn.lastSeen, sizeof(time_t), 1, fp) != sizeof(time_t), end_read, ret, -1);
            GOTO_IF(fread(&kn.stales, sizeof(int32_t), 1, fp) != sizeof(int32_t), end_read, ret, -1);

            buck->nodes.push_back(kn);
        }


        GOTO_IF(fwrite(&nodesCount, sizeof(int32_t), 1, fp) != sizeof(int32_t), end_read, ret, -1);
        for (j = 0; j < nodesCount; j++) {
            KadNode kn("");
            GOTO_IF(fread(kn.id.id, sizeof(unsigned char), KAD_ID_SZ, fp) != KAD_ID_SZ, end_read, ret, -1);
            GOTO_IF(fgetstdstr(kn.address, fp) == 0, end_read, ret, -1);
            GOTO_IF(fread(&kn.lastSeen, sizeof(time_t), 1, fp) != sizeof(time_t), end_read, ret, -1);
            GOTO_IF(fread(&kn.stales, sizeof(int32_t), 1, fp) != sizeof(int32_t), end_read, ret, -1);

            buck->replacementNodes.push_back(kn);
        }
    }

end_read:
    fclose(fp);

    return ret;
}

int KadRoutingTable::writeFile(FILE *fp)
{
    int ret, i, j, nodesCount;
    ret = 0;
    for (i = 0; i < KAD_ID_BIT_SZ; i++) {
        KadBucket *buck = this->buckets[i];
        // GOTO_IF(fwrite(&buck->height, sizeof(int32_t), 1, fp) != sizeof(int32_t), end_write, ret, -1);
        nodesCount = buck->nodes.size();
        GOTO_IF(fwrite(&nodesCount, sizeof(int32_t), 1, fp) != sizeof(int32_t), end_write, ret, -1);
        for (j = 0; j < nodesCount; j++) {
            KadNode kn = buck->nodes[i];
            GOTO_IF(fwrite(kn.id.id, sizeof(unsigned char), KAD_ID_SZ, fp) != KAD_ID_SZ, end_write, ret, -1);
            GOTO_IF(fwrite(kn.address.c_str(), kn.address.length() + 1, 1, fp) != kn.address.length() + 1, end_write, ret, -1);
            GOTO_IF(fwrite(&kn.lastSeen, sizeof(time_t), 1, fp) != sizeof(time_t), end_write, ret, -1);
            GOTO_IF(fwrite(&kn.stales, sizeof(int32_t), 1, fp) != sizeof(int32_t), end_write, ret, -1);
        }

        nodesCount = buck->replacementNodes.size();
        GOTO_IF(fwrite(&nodesCount, sizeof(int32_t), 1, fp) != sizeof(int32_t), end_write, ret, -1);
        for (j = 0; j < nodesCount; j++) {
            KadNode kn = buck->replacementNodes[i];
            GOTO_IF(fwrite(kn.id.id, sizeof(unsigned char), KAD_ID_SZ, fp) != KAD_ID_SZ, end_write, ret, -1);
            GOTO_IF(fwrite(kn.address.c_str(), kn.address.length() + 1, 1, fp) != kn.address.length() + 1, end_write, ret, -1);
            GOTO_IF(fwrite(&kn.lastSeen, sizeof(time_t), 1, fp) != sizeof(time_t), end_write, ret, -1);
            GOTO_IF(fwrite(&kn.stales, sizeof(int32_t), 1, fp) != sizeof(int32_t), end_write, ret, -1);
        }
    }

end_write:
    fclose(fp);

    return ret;
}

std::ostream &operator<<(std::ostream &os, const KadRoutingTable &kt2)
{
    os << "KadRoutingTable["
       << ", buckets=[\n";

    int i;
    for (i = 0; i < KAD_ID_BIT_SZ; i++) {
        os << *kt2.buckets[i] << "\n";
    }

    os << "]";
    return os;
}

std::vector<KadNode> KadRoutingTable::getAllNodes()
{
    int i;
    std::vector<KadNode> allNodes;
    for (i = 0; i < KAD_ID_BIT_SZ; i++) {
        std::vector<KadNode> on = this->buckets[i]->nodes;
        allNodes.insert(allNodes.end(), on.begin(), on.end());
    }
    return allNodes;
}
