#include "localnode.h"

#include "rpc/sdsrpcclient.h"
#include "common/logging.h"
#include "common/utils.h"
#include "simhash.h"

#include <set>
#include <algorithm>
#include <future>
#include <thread>

/*
    NodesLookupTask
*/
class NodesLookupTask : public SdsTask
{
public:
    NodesLookupTask(LocalNode *node_)
        : SdsTask(15*60), node(node_)
    {}

    ~NodesLookupTask() override = default;

protected:
    int execute() override
    {
        std::vector<KadId> toLook;
        int i;
        if (this->node->ktable->isFull()) {
            time_t now = time(nullptr);
            for (i = 0; i < KAD_ID_BIT_LENGTH; i++) {
                if ((now - this->node->ktable->getNodeAtHeight(i, 0).getLastSeen()) > UNIX_HOUR) {
                    toLook.push_back(this->node->ktable->getNodeAtHeight(i, rand() % KAD_BUCKET_MAX).getId());
                }
            }
        } else {
            for (i = 0; i < KAD_ID_BIT_LENGTH; i++) {
                toLook.push_back(KadId::idNbitsFarFrom(this->node->ktable->getSelfNode().getId(), i));
            }
        }
        for (i = 0; this->isRunning() && i < toLook.size(); i++) {
            this->doNodesLookup(toLook.at(i), false);
        }
        return 0;
    }

private:
    LocalNode *node;

    int doNodesLookup(const KadId targetId, bool check)
    {
        this->node->lock();

        const KadId selfNodeId = this->node->ktable->getSelfNode().getId();
        const std::string selfNodeAddress = this->node->ktable->getSelfNode().getAddress();
        static int ALPHA = 3;

        std::vector<KadNode> alphaClosest;
        this->node->ktable->getClosestTo(alphaClosest, targetId, ALPHA);

        this->node->unlock();

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
                if (ikn.getId() == selfNodeId) {
                    continue;
                }

                futures.push_back(std::move(std::async(std::launch::async, [ikn, selfNodeId, selfNodeAddress, targetId]() {
                    SdsRpcClient client(ikn.getAddress());
                    std::map<KadId, std::string> newNodes;

                    if (client.findNode(newNodes, selfNodeId, selfNodeAddress, targetId) != 0) {
                        newNodes.clear();
                    }
                    return newNodes;
                })));
            }

            discovered.clear();

            this->node->lock();
            for (int i = 0; i < futures.size(); i++) {
                KadNode ikn = alphaClosest[i];
                probed.insert(ikn.getId());

                std::map<KadId, std::string> newNodes = futures[i].get();
                if (newNodes.empty()) {
                    this->node->ktable->removeNode(ikn);
                    failed.insert(ikn.getId());
                } else {
                    this->node->ktable->pushNode(ikn);
                    for (auto it = newNodes.begin(); it != newNodes.end(); it++) {
                        if (probed.find(it->first) == probed.end()) {
                            nDiscovered++;
                            discovered.emplace_back(it->first, it->second);
                        }
                    }
                }
            }
            this->node->unlock();

            std::sort(discovered.begin(), discovered.end(), [targetId](const KadNode &a, const KadNode &b) {
                return (a.getId() - targetId) < (b.getId() - targetId);
            });

            alphaClosest.clear();
            for (auto it = discovered.begin(); it != discovered.end(); it++) {
                this->node->ktable->pushNode(*it);
                if (alphaClosest.size() < 3) {
                    alphaClosest.push_back(*it);
                }
            }

            if (time(nullptr) - startTime > 60) {
                break;
            }
        }

        logdebug << "Discovered " << nDiscovered << " closest nodes to " << targetId;
        return nDiscovered;
    }
};

/*
    PublishEntriesTask
*/

class EntriesPublishTask : public SdsTask
{
public:
    EntriesPublishTask(LocalNode *localNode_)
        : SdsTask(UNIX_HOUR), node(localNode_)
    {}

    ~EntriesPublishTask() override = default;

protected:
    int execute() override
    {
        std::vector<SearchEntry> results;

        this->node->lock();
        this->node->searchesDB->getEntriesForBroadcast(results);
        this->node->crawler->getEntriesForBroadcast(results);
        this->node->unlock();

        if (results.size() > 0)
            this->publishResults(results);

        return 0;
    }


private:
    LocalNode *node;

    void publishResults(const std::vector<SearchEntry> &results)
    {
        const KadId selfNodeId = this->node->ktable->getSelfNode().getId();
        const std::string selfNodeAddress = this->node->ktable->getSelfNode().getAddress();

        std::set<KadId> failed;
        std::set<KadNode> targetNodes;
        std::map<KadId, std::future<int>> futures;

        int i;
        for (auto rit = results.begin(); rit != results.end(); rit++) {

            this->node->lock();
            targetNodes.clear();
            std::vector<KadNode> nodes;
            this->node->ktable->getKClosestTo(nodes, rit->getSimHash().getId());
            for (auto nit = nodes.begin(); nit != nodes.end(); nit++) {
                KadId id = nit->getId();
                if (id == selfNodeId) {
                    SearchEntry se = *rit;
                    this->node->searchesDB->insertResult(se);
                } else {
                    targetNodes.insert(*nit);
                }
            }
            this->node->unlock();

            futures.clear();
            for (auto itn = targetNodes.begin(); itn != targetNodes.end(); itn++) {
                if (failed.find(itn->getId()) != failed.end()) {

                    futures[itn->getId()] = std::move(std::async(std::launch::async, [itn, selfNodeId, selfNodeAddress, rit]() {
                        SdsRpcClient client(itn->getAddress());
                        return client.storeResult(selfNodeId, selfNodeAddress, *rit);
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

        this->node->lock();
        for (auto it = failed.begin(); it != failed.end(); it++) {
            this->node->ktable->removeNode(*it);
        }
        this->node->unlock();
    }
};

/************************************************************************************************************************

  LocalNode

*************************************************************************************************************************/

LocalNode::LocalNode(SdsConfig &cfgs)
    : configs(cfgs)
{
    pthread_mutex_init(&this->mutex, nullptr);
    char path[1024];

    this->searchesDB = new SearchEntriesDB();
    sprintf(path, "%s/%s", cfgs.work_dir_path, "searches.db");
    this->searchesDB->open(path);

    this->ktable = new KadRoutingTable();
    sprintf(path, "%s/%s", cfgs.work_dir_path, "ktable.dat");
    if (this->ktable->readFile(path)) {
        logdebug << "ktable is empty populating with known nodes from configs";
        for (auto it = this->configs.known_peers.begin(); it != this->configs.known_peers.end(); it++) {
            KadNode kn(*it);
            this->ktable->pushNode(kn);
        }
    }

    this->nodesLookupTask = new NodesLookupTask(this);
    this->entriesPublishTask = new EntriesPublishTask(this);

    sprintf(path, "%s/%s", cfgs.work_dir_path, "crawlerseeds.dat");
    this->crawler = new WebCrawler(cfgs);
    if (this->crawler->load(path)) {
        logwarn << "crawler is not seeded";
    }
}

LocalNode::~LocalNode()
{
    pthread_mutex_destroy(&this->mutex);

    delete this->ktable;
    delete this->searchesDB;
    delete this->nodesLookupTask;
    delete this->entriesPublishTask;
}

void LocalNode::setSelfNodeAddress(std::string address)
{
    KadNode self(address.c_str());
    this->ktable->setSelfNode(self);

    logdebug << "self node " << self;
}

int LocalNode::ping(const KadId &id, std::string address)
{

    int res = 0;
    this->lock();

    KadNode kn(address);
    res = this->ktable->pushNode(kn);

    this->unlock();

    return res;
}

int LocalNode::findNode(std::map<KadId, std::string> &nearest, const KadId &id)
{
    this->lock();

    std::vector<KadNode> nodes;
    this->ktable->getKClosestTo(nodes, id);

    int i;
    nearest.clear();
    for (i = 0; i < nodes.size(); i++) {
        KadNode kn = nodes[i];
        nearest[kn.getId()] = kn.getAddress();
    }

    this->unlock();

    return nearest.size();
}

int LocalNode::storeResult(SearchEntry se)
{
    this->lock();

    this->searchesDB->insertResult(se);

    this->unlock();

    return 0;
}

int LocalNode::findResults(std::vector<SearchEntry> &results, const char *query)
{
    this->lock();

    results.clear();
    this->searchesDB->doSearch(results, query);

    this->unlock();

    return results.size();
}

int LocalNode::nodeConnected(const KadId &id, std::string &address)
{
    this->lock();

    KadNode kn(id, address);
    loginfo << "new neighbour node conected " << kn;
    this->ktable->pushNode(kn);

    this->unlock();

    return 0;
}

int LocalNode::doSearch(std::vector<SearchEntry> &results, const char *query)
{
    this->lock();

    const KadId selfNodeId = this->ktable->getSelfNode().getId();
    const std::string selfNodeAddress = this->ktable->getSelfNode().getAddress();

    int i;
    std::set<KadId> probed = {};
    std::set<KadId> failed;
    std::vector<KadNode> nodes;
    std::set<KadNode> targetNodes;
    SimHash simHash(query);
    this->ktable->getKClosestTo(nodes, simHash.getId());
    for (auto it = nodes.begin(); it != nodes.end(); it++) {
        KadId id = it->getId();
        if (id == selfNodeId) {
            this->searchesDB->doSearch(results, query);
        } else {
            targetNodes.insert(*it);
        }
    }

    this->unlock();

    std::map<KadId, std::future<std::pair<int, std::vector<SearchEntry>>>> futures;
    for (auto ikn = targetNodes.begin(); ikn != targetNodes.end(); ikn++) {
        KadId id = ikn->getId();
        if (std::find(failed.begin(), failed.end(), id) != failed.end()) {
            continue;
        }

        futures[id] = std::move(std::async(std::launch::async, [ikn, selfNodeId, selfNodeAddress ,query] () {
            SdsRpcClient client(ikn->getAddress());
            std::vector<SearchEntry> newResults;
            int res = client.findResults(newResults, selfNodeId, selfNodeAddress, query);
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

    this->lock();

    for (auto it = failed.begin(); it != failed.end(); it++) {
        this->ktable->removeNode(*it);
    }

    this->unlock();

    /*
        If no results are found then fallback to the external centralized search engines
    */
    if (results.size() == 0) {
        this->crawler->doSearch(results, query);
    }

    return results.size();
}

void LocalNode::startTasks()
{
    this->nodesLookupTask->start();
    this->entriesPublishTask->start();
    this->crawler->startCrawling();
}

void LocalNode::shutdown()
{
    loginfo << "stopping tasks...";
    this->nodesLookupTask->stop();
    this->entriesPublishTask->stop();
    this->crawler->stopCrawling();

    this->lock();

    char path[1024];
    this->searchesDB->close();

    sprintf(path, "%s/%s", this->configs.work_dir_path, "ktable.dat");
    this->ktable->writeFile(path);

    sprintf(path, "%s/%s", this->configs.work_dir_path, "crawlerseeds.dat");
    this->crawler->save(path);

    this->unlock();
}

void LocalNode::lock()
{
    pthread_mutex_lock(&this->mutex);
}

void LocalNode::unlock()
{
    pthread_mutex_unlock(&this->mutex);
}
