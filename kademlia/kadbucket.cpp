#include <string.h>
#include <algorithm>

#include "kadbucket.h"

/*
    KadBucket
*/

KadBucket::KadBucket(int height)
    : height(height)
{}

KadBucket::~KadBucket()
{

}

bool KadBucket::hasNode(const KadNode &kn)
{
    return std::find(this->nodes.begin(), this->nodes.end(), kn) != this->nodes.end();
}

bool KadBucket::pushNode(const KadNode &kn)
{
    auto ikn = std::find(this->nodes.begin(), this->nodes.end(), kn);
    bool found = ikn != this->nodes.end();
    if (found) {
        ikn->seenNow();
        ikn->resetStales();
    } else if (this->nodes.size() < KAD_BUCKET_MAX) {
        this->nodes.push_back(kn);
    } else {
        int staleMax = 0;
        auto stalest = this->nodes.end();
        for (auto it = this->nodes.begin(); it < this->nodes.end(); it++) {
            if (it->getStales() >= STALES_THR && it->getStales() > staleMax) {
                staleMax = it->getStales();
                stalest = it;
            }
        }

        if (stalest != this->nodes.end()) {
            this->nodes.erase(stalest);
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

    this->reorder();
    return found;
}

bool KadBucket::removeNode(KadNode &kn)
{
    return this->removeNode(kn.getId());
}

bool KadBucket::removeNode(const KadId &id)
{
    auto ikn = std::find_if(this->nodes.begin(), this->nodes.end(), [id] (const KadNode &kn1) {
        return kn1.getId() == id;
    });

    if (ikn != this->nodes.end()) {
        if (this->replacementNodes.size()) {
            this->nodes.erase(ikn);

            auto first = this->replacementNodes.begin();
            this->nodes.push_back(*first);
            this->replacementNodes.erase(first);
            this->reorder();
        } else {
            ikn->incrementStales();
        }
        return true;
    }
    return false;
}

int KadBucket::getHeight() const
{
    return this->height;
}

bool KadBucket::isFull()
{
    return this->nodes.size() == KAD_BUCKET_MAX;
}

size_t KadBucket::getNodesCount()
{
    return this->nodes.size();
}

size_t KadBucket::getReplacementCount()
{
    return this->replacementNodes.size();
}

void KadBucket::reorder()
{
    std::sort(this->nodes.begin(), this->nodes.end());
    std::sort(this->replacementNodes.begin(), this->replacementNodes.end());
}

std::ostream &operator<<(std::ostream &os, const KadBucket &kb2)
{
    os << "KadBucket["
       << "height=" << kb2.height
       << ", nodes=[\n";

    for (auto it = kb2.nodes.begin(); it != kb2.nodes.end(); it++) {
        os << *it << "\n";
    }

    os << "]\n, replacementNodes=[\n";
    for (auto it = kb2.replacementNodes.begin(); it != kb2.replacementNodes.end(); it++) {
        os << *it << "\n";
    }

    os << "]\n]";
    return os;
}
