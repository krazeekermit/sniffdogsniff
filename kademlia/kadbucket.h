#ifndef KADBUCKET_H
#define KADBUCKET_H

#include "kadnode.h"

#include <vector>

#define KAD_BUCKET_MAX 20
#define STALES_THR     5

class KadBucket
{
    friend class KadRoutingTable;

public:
    KadBucket(int height);
    ~KadBucket();

    bool pushNode(const KadNode &kn);
    bool removeNode(KadNode &kn);
    bool removeNode(const KadId &id);

    friend std::ostream &operator<<(std::ostream &os, const KadBucket &kb2);

    int getHeight() const;
    bool isFull();

    size_t getNodesCount();
    size_t getReplacementCount();

    KadNode getNode(const KadId id) const;
    KadNode getReplacement(const KadId id) const;

private:
    int height;
    std::vector<KadNode> nodes;
    std::vector<KadNode> replacementNodes;

    void reorder();
//    getNode(KadNode *kn);
//    KadNode* getReplacementNode(KadNode *kn);
};

#endif // KADBUCKET_H
