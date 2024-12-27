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

    bool pushNode(KadNode &kn);
    bool removeNode(KadNode &kn);

    int getKClosestTo(std::vector<KadNode> &nodes, const KadId &id);
    int getClosestTo(std::vector<KadNode> &nodes, const KadId &id, int count);

    int readFile(FILE *fp);
    int writeFile(FILE *fp);

    friend std::ostream &operator<<(std::ostream &os, const KadRoutingTable &kt2);

private:
    KadNode selfNode;
    KadBucket **buckets;

    std::vector<KadNode> getAllNodes();
};

#endif // KADEMLIAROUTINGTABLE_H
