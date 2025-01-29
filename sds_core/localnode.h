#ifndef LOCALNODE_H
#define LOCALNODE_H

#include "sds_config.h"

#include "crawler/webcrawler.h"
#include "kademlia/kadroutingtable.h"
#include "sdstask.h"

#include <pthread.h>
#include <map>
#include <vector>

class NodesLookupTask;

class LocalNode
{
friend class NodesLookupTask;
friend class EntriesPublishTask;

public:
    LocalNode(SdsConfig &cfgs);
    ~LocalNode();

    void setSelfNodeAddress(std::string address);
    int ping(const KadId &id, std::string address);
    int findNode(std::map<KadId, std::string> &nearest, const KadId &id);
    int storeResult(SearchEntry se);
    int findResults(std::vector<SearchEntry> &results, const char *query);
    int doSearch(std::vector<SearchEntry> &results, const char *query);

    // used to insert new connected node into the ktable
    // usually called by the rpc request handler
    int nodeConnected(const KadId &id, std::string &address);
    // checkNode(id kademlia.KadId, addr string) bool

    void startTasks();
    void shutdown();

private:
    SdsConfig configs;
    pthread_mutex_t mutex;
    KadRoutingTable *ktable;
    SearchEntriesDB *searchesDB;
    WebCrawler *crawler;
    SdsTask *nodesLookupTask;
    SdsTask *entriesPublishTask;

    void lock();
    void unlock();
};

#endif // LOCALNODE_H
