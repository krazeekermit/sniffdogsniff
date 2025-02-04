#ifndef KADEMLIAROUTINGTABLE_H
#define KADEMLIAROUTINGTABLE_H

#include "kadbucket.h"

#include <vector>

class KadRoutingTable
{
public:
    KadRoutingTable();
    ~KadRoutingTable();

    KadNode getSelfNode() const;
    void setSelfNode(KadNode &newSelfNode);

    bool isFull();
    bool isEmpty();

    bool pushNode(const KadNode &kn);
    bool removeNode(KadNode &kn);
    bool removeNode(const KadId &id);

    int getKClosestTo(std::vector<KadNode> &nodes, const KadId &id);
    int getClosestTo(std::vector<KadNode> &nodes, const KadId &id, int count);
    const KadNode &getNodeAtHeight(int height, int index);

    int readFile(const char *path);
    int writeFile(const char *path);

    friend std::ostream &operator<<(std::ostream &os, const KadRoutingTable &kt2);

private:
    KadNode selfNode;
    KadBucket **buckets;

    std::vector<KadNode> getAllNodes();
};

#endif // KADEMLIAROUTINGTABLE_H
