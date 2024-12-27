#include "localnode.h"

#include "rpc/sdsrpcclient.h"
#include "logging.hpp"

#include <set>
#include <algorithm>
#include <future>
#include <thread>

LocalNode::LocalNode(SdsConfig &cfgs)
{
    pthread_mutex_init(&this->mutex, nullptr);
}

int LocalNode::ping(const KadId &id, std::string address)
{

    int res = 0;
    pthread_mutex_lock(&this->mutex);

    KadNode kn(address);
    res = this->ktable.pushNode(kn);

    pthread_mutex_unlock(&this->mutex);

    return res;
}

int LocalNode::findNode(std::map<KadId, std::string> &nearest, const KadId &id)
{
    pthread_mutex_lock(&this->mutex);

    std::vector<KadNode> nodes;
    this->ktable.getKClosestTo(nodes, id);

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

    this->searchesDB.insertResult(se);

    pthread_mutex_unlock(&this->mutex);

    return 0;
}

int LocalNode::findResults(std::vector<SearchEntry> &results, const char *query)
{
    pthread_mutex_lock(&this->mutex);

    results.clear();
    this->searchesDB.doSearch(results, query);

    pthread_mutex_unlock(&this->mutex);

    return results.size();
}

//************************************************************************************//
int LocalNode::doNodesLookup(KadNode &target, bool check)
{
    const KadId targetId = target.getId();
    const KadId selfNodeId = this->ktable.getSelfNode().getId();
    static int ALPHA = 3;

    std::vector<KadNode> alphaClosest;
    if (this->ktable.getClosestTo(alphaClosest, targetId, ALPHA))
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
                this->ktable.removeNode(ikn);
                failed.insert(ikn.getId());
            } else {
                this->ktable.pushNode(ikn);
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
            this->ktable.pushNode(*it);
            if (alphaClosest.size() < 3) {
                alphaClosest.push_back(*it);
            }
        }

//        if (time(nullptr) - startTime > TIME_TASK_MAX)
//            break;
    }

    logdebug(<< "Discovered " << nDiscovered << " closest nodes to " << targetId);
    return nDiscovered;
}

//************************************************************************************//
void LocalNode::publishResults(const std::vector<SearchEntry> &results)
{
    int i;
    const KadId selfNodeId = this->ktable.getSelfNode().getId();
//failed := kademlia.NewKNodesMap()
    std::set<KadNode> failed;
    std::set<KadNode> targetNodes;
//	var wg sync.WaitGroup
    std::vector<std::future<std::pair<KadNode, int>>> futures;

//	toInsertSelf := make([]SearchResult, 0)

//	for _, sr := range results {
    for (auto rit = results.begin(); rit != results.end(); rit++) {
//		qn := len(sr.QueryMetrics)
        int qn = METRICS_LEN;

        targetNodes.clear();
//		nodes := make(map[kademlia.KadId]*kademlia.KNode, 0)
//		ln.tsLock.Lock()
        //		for i := 0; i < qn; i++ {
        for (i = 0; i < qn; i++) {
            //			for _, ikn := range ln.ktable.GetNClosestTo(sr.QueryMetrics[i], kademlia.K/qn) {
            //				nodes[ikn.Id] = ikn
            //			}
            //		}

            std::vector<KadNode> nodes;
            this->ktable.getClosestTo(nodes, rit->getMetrics()[i], KAD_BUCKET_MAX / qn);
            for (auto nit = nodes.begin(); nit != nodes.end(); nit++) {
                KadId id = nit->getId();
                if (id == selfNodeId) {
                    this->searchesDB.insertResult(*rit);
                } else {
                    targetNodes.insert(*nit);
                }
            }
        }
//		ln.tsLock.Unlock()

        futures.clear();
        for (auto itn = targetNodes.begin(); itn != targetNodes.end(); itn++) {
            if (failed.find(*itn) != failed.end()) {
                //			_, present := failed.Get(ikn.Id)
                //			if present {
                //				// if it is a failed node avoid to contact it again
                //				continue
                //			}
                //			if ikn.Id.Eq(ln.SelfNode().Id) {
                //				toInsertSelf = append(toInsertSelf, sr)
                //				continue
                //			}

                //			wg.Add(1)
                //			wgCount++
                //			go func(kn, source *kademlia.KNode, value SearchResult, failed *kademlia.KNodesMap) {
                //				defer wg.Done()

                //				rn := NewNodeClient(kn.Address)
                //				err := rn.StoreResult(sr, source)
                //				if err != nil {
                //					failed.Put(kn)
                //				}
                //			}(ikn, ln.ktable.SelfNode(), sr, failed)
                futures.push_back(std::move(std::async(std::launch::async, [itn, rit]() {
                    SdsRpcClient client(itn->getAddress());
                    int ret = client.storeResult(*rit);
                    return std::make_pair(*itn, ret);
                })));

                //			if wgCount >= kademlia.ALPHA {
                //				wgCount = 0
                //				wg.Wait()
                //			}

                if (futures.size() >= 3) {
                    for (auto fit = futures.begin(); fit != futures.end(); fit++) {
                        std::pair<KadNode, int> r = fit->get();
                        if (r.second != 0)
                            failed.insert(r.first);

                        futures.clear();
                    }
                }
            }
        }
    }

//	ln.tsLock.Lock()
//	for _, sr := range toInsertSelf {
//		ln.searchDB.InsertResult(sr)
//	}

    for (auto it = failed.begin(); it != failed.end(); it++) {
        KadNode kn = *it;
        this->ktable.removeNode(kn);
    }

//	for _, kid := range failed.Keys() {
//		kn, _ := failed.Get(kid)
//		ln.ktable.RemoveNode(kn)
//	}
//	ln.tsLock.Unlock()
}
