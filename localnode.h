#ifndef LOCALNODE_H
#define LOCALNODE_H

#include "sds_config.h"

#include "searchentriesdb.h"
#include "kademlia/kadroutingtable.h"

#include <pthread.h>
#include <map>
#include <vector>

class LocalNode
{
public:
    LocalNode(SdsConfig &cfgs);
    ~LocalNode();

    int ping(const KadId &id, std::string address);
    int findNode(std::map<KadId, std::string> &nearest, const KadId &id);
    int storeResult(SearchEntry se);
    int findResults(std::vector<SearchEntry> &results, const char *query);

    // used to insert new connected node into the ktable
    // usually called by the rpc request handler
    int nodeConnected(const unsigned char *id, const char *address);
//    checkNode(id kademlia.KadId, addr string) bool

private:
    pthread_mutex_t mutex;
    KadRoutingTable ktable;
    SearchEntriesDB searchesDB;

    int doNodesLookup(KadNode &target, bool check);
    void publishResults(const std::vector<SearchEntry> &results);
};

#endif // LOCALNODE_H
