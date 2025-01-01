#include "localnode.h"

#include "rpc/sdsrpcclient.h"
#include "logging.hpp"
#include "searchengine.h"
#include "utils.h"

#include <set>
#include <algorithm>
#include <future>
#include <thread>

LocalNode::LocalNode(SdsConfig &cfgs)
    : configs(cfgs)
{
    pthread_mutex_init(&this->mutex, nullptr);
    this->ktable = new KadRoutingTable();
    for (auto it = this->configs.known_peers.begin(); it != this->configs.known_peers.end(); it++) {
        KadNode kn(*it);
        this->ktable->pushNode(kn);
    }

    this->searchesDB = new SearchEntriesDB();
}

LocalNode::~LocalNode()
{
    delete this->ktable;
    delete this->searchesDB;
    delete this->syncNodesTask;
    delete this->broadcastResultsTask;
}

int LocalNode::ping(const KadId &id, std::string address)
{

    int res = 0;
    pthread_mutex_lock(&this->mutex);

    KadNode kn(address);
    res = this->ktable->pushNode(kn);

    pthread_mutex_unlock(&this->mutex);

    return res;
}

int LocalNode::findNode(std::map<KadId, std::string> &nearest, const KadId &id)
{
    pthread_mutex_lock(&this->mutex);

    std::vector<KadNode> nodes;
    this->ktable->getKClosestTo(nodes, id);

    int i;
    nearest.clear();
    for (i = 0; i < nodes.size(); i++) {
        KadNode kn = nodes[i];
        nearest[kn.getId()] = kn.getAddress();
    }

    pthread_mutex_unlock(&this->mutex);

    return nearest.size();
}

int LocalNode::storeResult(SearchEntry se)
{
    pthread_mutex_lock(&this->mutex);

    this->searchesDB->insertResult(se);

    pthread_mutex_unlock(&this->mutex);

    return 0;
}

int LocalNode::findResults(std::vector<SearchEntry> &results, const char *query)
{
    pthread_mutex_lock(&this->mutex);

    results.clear();
    this->searchesDB->doSearch(results, query);

    pthread_mutex_unlock(&this->mutex);

    return results.size();
}

int LocalNode::doSearch(std::vector<SearchEntry> &results, const char *query)
{
    const KadId selfNodeId = this->ktable->getSelfNode().getId();

    int i;
    KadId metrics[METRICS_LEN];
    std::set<KadId> probed = {};
    std::set<KadId> failed;
    std::vector<KadNode> nodes;
    std::set<KadNode> targetNodes;
    SearchEntry::evaluateMetrics(metrics, query);
    for (i = 0; i < METRICS_LEN; i++) {
        this->ktable->getKClosestTo(nodes, metrics[i]);
        for (auto it = nodes.begin(); it != nodes.end(); it++) {
            KadId id = it->getId();
            if (id == selfNodeId) {
                this->searchesDB->doSearch(results, query);
            } else {
                targetNodes.insert(*it);
            }
        }
    }

    std::map<KadId, std::future<std::pair<int, std::vector<SearchEntry>>>> futures;
    for (auto ikn = targetNodes.begin(); ikn != targetNodes.end(); ikn++) {
        KadId id = ikn->getId();
        if (std::find(failed.begin(), failed.end(), id) != failed.end()) {
            continue;
        }

        futures[id] = std::move(std::async(std::launch::async, [ikn, query] () {
            SdsRpcClient client(ikn->getAddress());
            std::vector<SearchEntry> newResults;
            int res = client.findResults(newResults, query);
            return std::make_pair(res, newResults);
        }));

        if (futures.size() >= 3) {
            for (auto fit = futures.begin(); fit != futures.end(); fit++) {
                std::pair<int, std::vector<SearchEntry>> r = fit->second.get();
                if (r.first != 0) {
                    failed.insert(fit->first);
                } else {
                    results.insert(results.end(), r.second.begin(), r.second.end());
                }
            }
            futures.clear();
        }
    }

    for (auto it = failed.begin(); it != failed.end(); it++) {
        this->ktable->removeNode(*it);
    }

    //do search ...... on external search engines replace with crawler
    for (auto it = this->configs.search_engines.begin(); it != this->configs.search_engines.end(); it++) {
        SearchEngine en = SearchEngine(*it);
        en.doSearch(results, query);
    }

    std::async(std::launch::async, [this, results] () {
        this->publishResults(results);
    });

    return results.size();
}

void LocalNode::startTasks()
{
    //Node sync task
    this->syncNodesTask = new SdsTask([this] () {
        //this->doNodesLookup();
        return;
    }, 1000);

    //Results publish task
    this->broadcastResultsTask = new SdsTask([this] () {
        std::vector<SearchEntry> results;
        if (this->searchesDB->getEntriesForBroadcast(results) > 0)
            this->publishResults(results);

    }, UNIX_HOUR);

}

//************************************************************************************//
int LocalNode::doNodesLookup(KadNode &target, bool check)
{
    const KadId targetId = target.getId();
    const KadId selfNodeId = this->ktable->getSelfNode().getId();
    static int ALPHA = 3;

    std::vector<KadNode> alphaClosest;
    if (this->ktable->getClosestTo(alphaClosest, targetId, ALPHA))
        return 0;


    int nDiscovered = 0;
    long startTime = time(nullptr);

    int i;
    std::vector<std::future<std::map<KadId, std::string>>> futures;
    std::vector<KadNode> discovered;
    std::set<KadId> probed = {};
    std::set<KadId> failed;

    while (alphaClosest.size()) {
        futures.clear();
        for (int i = 0; i < alphaClosest.size(); i++) {
            KadNode ikn = alphaClosest[i];
            if (ikn.getId() == selfNodeId)
                continue;

            futures.push_back(std::move(std::async(std::launch::async, [ikn, targetId]() {
                SdsRpcClient client(ikn.getAddress());
                std::map<KadId, std::string> newNodes;

                if (client.findNode(newNodes, targetId) != 0) {
                    newNodes.clear();
                }
                return newNodes;
            })));
        }

        discovered.clear();
        for (int i = 0; i < alphaClosest.size(); i++) {
            KadNode ikn = alphaClosest[i];
            probed.insert(ikn.getId());

            std::map<KadId, std::string> newNodes = futures[i].get();
            if (newNodes.empty()) {
                this->ktable->removeNode(ikn);
                failed.insert(ikn.getId());
            } else {
                this->ktable->pushNode(ikn);
                for (auto it = newNodes.begin(); it != newNodes.end(); it++) {
                    if (probed.find(it->first) == probed.end()) {
                        nDiscovered++;
                        discovered.emplace_back(it->first, it->second);
                    }
                }
            }
        }

        std::sort(discovered.begin(), discovered.end(), [targetId](const KadNode &a, const KadNode &b) {
            return (a.getId() - targetId) < (b.getId() - targetId);
        });

        alphaClosest.clear();
        for (auto it = discovered.begin(); it != discovered.end(); it++) {
            this->ktable->pushNode(*it);
            if (alphaClosest.size() < 3) {
                alphaClosest.push_back(*it);
            }
        }

//        if (time(nullptr) - startTime > TIME_TASK_MAX)
//            break;
    }

    logdebug << "Discovered " << nDiscovered << " closest nodes to " << targetId;
    return nDiscovered;
}

//************************************************************************************//
void LocalNode::publishResults(const std::vector<SearchEntry> &results)
{
    int i;
    const KadId selfNodeId = this->ktable->getSelfNode().getId();
    std::set<KadId> failed;
    std::set<KadNode> targetNodes;
    std::map<KadId, std::future<int>> futures;

    for (auto rit = results.begin(); rit != results.end(); rit++) {
        int qn = METRICS_LEN;

        targetNodes.clear();
        for (i = 0; i < qn; i++) {
            std::vector<KadNode> nodes;
            this->ktable->getClosestTo(nodes, rit->getMetrics()[i], KAD_BUCKET_MAX / qn);
            for (auto nit = nodes.begin(); nit != nodes.end(); nit++) {
                KadId id = nit->getId();
                if (id == selfNodeId) {
                    this->searchesDB->insertResult(*rit);
                } else {
                    targetNodes.insert(*nit);
                }
            }
        }
//		ln.tsLock.Unlock()

        futures.clear();
        for (auto itn = targetNodes.begin(); itn != targetNodes.end(); itn++) {
            if (failed.find(itn->getId()) != failed.end()) {

                futures[itn->getId()] = std::move(std::async(std::launch::async, [itn, rit]() {
                    SdsRpcClient client(itn->getAddress());
                    return client.storeResult(*rit);
                }));

                if (futures.size() >= 3) {
                    for (auto fit = futures.begin(); fit != futures.end(); fit++) {
                        int ret = fit->second.get();
                        if (ret != 0)
                            failed.insert(fit->first);
                    }
                    futures.clear();
                }
            }
        }
    }

    for (auto it = failed.begin(); it != failed.end(); it++) {
        this->ktable->removeNode(*it);
    }
}
