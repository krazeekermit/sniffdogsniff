package core

import (
	"errors"
	"math/rand"
	"path/filepath"
	"sync"
	"time"

	"github.com/sniffdogsniff/kademlia"
	"github.com/sniffdogsniff/logging"
	"github.com/sniffdogsniff/proxies"
	"github.com/sniffdogsniff/util"
)

const FIRST_SYNC_LOCK_FILE_NAME = "firstsync.lock"

const LOOKUP_ROUND_TIME_THR = 20        // seconds
const NORMAL_NODES_LOOKUP_DELAY = 900   // seconds
const NON_FULL_NODES_LOOKUP_DELAY = 120 // seconds
const NORMAL_SYNC_DELAY = 30            // seconds
const INITIAL_DELAY = 1                 // seconds

const SELF_PEER_FILE_NAME = "selfnode.dat"

/* Maximum lenght of searches that a peer can send per time */
const MAX_SYNC_SIZE = 104857600 / SEARCH_RESULT_BYTE_SIZE // 100 MBytes / 512 bytes

var CANT_SYNC_ERROR = errors.New("first sync, cant sync from this")

type NodeInterface interface {
	Ping(id kademlia.KadId, address string) error
	FindNode(id kademlia.KadId) map[kademlia.KadId]string
	StoreResult(sr SearchResult)
	FindResults(query string) []SearchResult

	// used to insert new connected node into the ktable
	// usually called by the rpc request handler
	NodeConnected(id kademlia.KadId, addr string)
}

type LocalNode struct {
	proxySettings    proxies.ProxySettings
	canInvalidate    bool
	tsLock           *sync.Mutex // tread safe access from different threads, the NodeServer the WebUi, the SyncWithPeers()
	searchDB         SearchDB
	searchEngines    map[string]SearchEngine
	minResultsThr    int
	selfNodeFilePath string
	ktable           *kademlia.KadRoutingTable
	knownNodes       map[kademlia.KadId]string
}

func NewLocalNode(configs SdsConfig) *LocalNode {
	ln := LocalNode{}
	ln.knownNodes = configs.KnownPeers
	ln.selfNodeFilePath = filepath.Join(configs.WorkDirPath, SELF_PEER_FILE_NAME)
	ln.ktable = kademlia.NewKadRoutingTable()
	ln.ktable.Open(configs.WorkDirPath)

	ln.searchDB.Open(configs.WorkDirPath, configs.searchDBMaxCacheSize, 3600*24)
	ln.proxySettings = configs.ProxySettings
	ln.canInvalidate = configs.AllowResultsInvalidation
	ln.tsLock = &sync.Mutex{}
	ln.searchEngines = configs.searchEngines
	ln.minResultsThr = 10 // 10 placeholder number will be defined in SdsConfigs
	return &ln
}

func (ln *LocalNode) SelfNode() *kademlia.KNode {
	return ln.ktable.SelfNode()
}

func (ln *LocalNode) SetNodeAddress(addr string) {
	ln.tsLock.Lock()
	ln.ktable.SetSelfNode(kademlia.NewKNode(kademlia.NewKadId(addr), addr))
	if ln.ktable.IsEmpty() {
		for id, addr := range ln.knownNodes {
			logging.LogTrace(addr)
			ln.ktable.PushNode(kademlia.NewKNode(id, addr))
		}
	}
	// ln.ktable.Flush() // is needed?
	ln.tsLock.Unlock()
}

func (ln *LocalNode) KadRoutingTable() *kademlia.KadRoutingTable {
	return ln.ktable
}

func (ln *LocalNode) FindNode(id kademlia.KadId) map[kademlia.KadId]string {
	ln.tsLock.Lock()
	closest := ln.ktable.GetKClosestTo(id)
	ln.tsLock.Unlock()

	closestMap := make(map[kademlia.KadId]string)
	for _, ikn := range closest {
		closestMap[ikn.Id] = ikn.Address
	}
	return closestMap
}

func (ln *LocalNode) StoreResult(sr SearchResult) {
	ln.tsLock.Lock()
	ln.searchDB.InsertResult(sr)
	ln.tsLock.Unlock()
}

func (ln *LocalNode) Ping(id kademlia.KadId, addr string) error {
	ln.tsLock.Lock()
	ln.ktable.PushNode(kademlia.NewKNode(id, addr))
	ln.tsLock.Unlock()
	return nil
}

func (ln *LocalNode) NodeConnected(id kademlia.KadId, addr string) {
	ln.tsLock.Lock()
	ln.ktable.PushNode(kademlia.NewKNode(id, addr))
	ln.tsLock.Unlock()
}

func (ln *LocalNode) InsertSearchResult(sr SearchResult) {
	ln.PublishResults([]SearchResult{sr})
	ln.tsLock.Lock()
	ln.searchDB.InsertResult(sr)
	ln.tsLock.Unlock()
}

func (ln *LocalNode) InvalidateSearchResult(h Hash256) {
}

func (ln *LocalNode) UpdateResultScore(hash string) {
}

func (ln *LocalNode) FindResults(query string) []SearchResult {
	ln.tsLock.Lock() // tread safe access to SearchDB
	results := ln.searchDB.DoSearch(query)
	ln.tsLock.Unlock()
	return results
}

func (ln *LocalNode) DoSearch(query string) []SearchResult {
	nodes := make(map[kademlia.KadId]*kademlia.KNode, 0) // avoid duplicates
	ln.tsLock.Lock()
	for _, m := range EvalQueryMetrics(query) {
		for _, ikn := range ln.ktable.GetKClosestTo(m) {
			nodes[ikn.Id] = ikn
		}
	}
	ln.tsLock.Unlock()

	var wg sync.WaitGroup
	var failed, emptyProbed, results sync.Map

	wgCount := 0
	for _, ikn := range nodes {
		_, present := failed.Load(ikn.Id)
		if present {
			// if it is a failed node avoid to contact it again
			continue
		}
		if ikn.Id.Eq(ln.SelfNode().Id) {
			for _, v := range ln.FindResults(query) {
				results.LoadOrStore(v.ResultHash, v)
			}
			continue
		}

		wg.Add(1)
		wgCount++
		go func(kn, source *kademlia.KNode, query string, failed, emptyProbed, results *sync.Map) {
			defer wg.Done()

			rn := NewNodeClient(kn.Address, ln.proxySettings)
			vals, err := rn.FindResults(query, source)
			if err != nil {
				failed.Store(ikn.Id, kn)
				return
			}
			if len(vals) > 0 {
				for _, v := range vals {
					results.LoadOrStore(v.ResultHash, v)
				}
			} else {
				emptyProbed.LoadOrStore(kn.Id, kn)
			}
		}(ikn, ln.ktable.SelfNode(), query, &failed, &emptyProbed, &results)

		if wgCount >= kademlia.ALPHA {
			wgCount = 0
			wg.Wait()
		}
	}

	rSlice := make([]SearchResult, 0)
	results.Range(func(key, value any) bool {
		sr, ok := value.(SearchResult)
		if ok {
			rSlice = append(rSlice, sr)
		}
		return true
	})

	ln.tsLock.Lock()
	failed.Range(func(key, value any) bool {
		kn, ok := value.(*kademlia.KNode)
		if !ok {
			return true
		}
		ln.ktable.RemoveNode(kn)
		return true
	})
	ln.tsLock.Unlock()

	if len(rSlice) == 0 {
		/*
			If no values were found in any nodes then rely on the centralized external
			search engines. The found results are stored in the network.
		*/
		rSlice = append(rSlice, DoParallelSearchOnExtEngines(ln.searchEngines, query)...)
		ln.PublishResults(rSlice)
		return rSlice
	}

	// re stores the values on nodes that are supposed to have the value but does not have it
	emptyProbed.Range(func(key, value any) bool {
		kn, ok := value.(*kademlia.KNode)
		if ok {
			rn := NewNodeClient(kn.Address, ln.proxySettings)
			for _, sr := range rSlice {
				rn.StoreResult(sr, ln.SelfNode())
			}
		}
		return true
	})

	return rSlice
}

/*
	Publish Results
*/

func (ln *LocalNode) PublishResults(results []SearchResult) {
	var failed sync.Map
	var wg sync.WaitGroup

	toInsertSelf := make([]SearchResult, 0)

	for _, sr := range results {
		qn := len(sr.QueryMetrics)

		nodes := make(map[kademlia.KadId]*kademlia.KNode, 0)
		ln.tsLock.Lock()
		for i := 0; i < qn; i++ {
			for _, ikn := range ln.ktable.GetNClosestTo(sr.QueryMetrics[i], kademlia.K/qn) {
				nodes[ikn.Id] = ikn
			}
		}
		ln.tsLock.Unlock()

		wgCount := 0
		for _, ikn := range nodes {
			_, present := failed.Load(ikn.Id)
			if present {
				// if it is a failed node avoid to contact it again
				continue
			}
			if ikn.Id.Eq(ln.SelfNode().Id) {
				toInsertSelf = append(toInsertSelf, sr)
				continue
			}

			wg.Add(1)
			wgCount++
			go func(kn, source *kademlia.KNode, value SearchResult, failed *sync.Map) {
				defer wg.Done()

				rn := NewNodeClient(kn.Address, ln.proxySettings)
				err := rn.StoreResult(sr, source)
				if err != nil {
					failed.Store(kn.Id, kn)
				}
			}(ikn, ln.ktable.SelfNode(), sr, &failed)

			if wgCount >= kademlia.ALPHA {
				wgCount = 0
				wg.Wait()
			}

		}
	}

	ln.tsLock.Lock()
	for _, sr := range toInsertSelf {
		ln.searchDB.InsertResult(sr)
	}

	failed.Range(func(key, value any) bool {
		kn, ok := value.(*kademlia.KNode)
		if !ok {
			return true
		}
		ln.ktable.RemoveNode(kn)
		return true
	})
	ln.tsLock.Unlock()
}

func (ln *LocalNode) StartPublishTask() {
	ticker := time.NewTicker(INITIAL_DELAY * time.Second)
	go func() {
		for range ticker.C {
			ticker.Reset(time.Duration(util.TIME_HOUR_UNIX) * time.Second)
			ln.tsLock.Lock()
			results := ln.searchDB.ResultsToPublish()
			logging.LogInfo("Publishing", len(results), "results...")
			ln.tsLock.Unlock()
			ln.PublishResults(results)
		}
	}()
}

/*
	Node Lookup
*/

func (ln *LocalNode) DoNodesLookup(targetNode *kademlia.KNode) int { // not yet operational
	alphaClosest := ln.ktable.GetNClosestTo(targetNode.Id, kademlia.ALPHA)
	if len(alphaClosest) == 0 {
		return 0
	}

	nDiscovered := 0
	startTime := util.CurrentUnixTime()

	var closest *kademlia.KNode = alphaClosest[0]
	var wg sync.WaitGroup
	var discovered, probed, failed sync.Map

	self := ln.SelfNode()
	probed.Store(self.Id, self.Address)

	for {
		for _, ikn := range alphaClosest {
			wg.Add(1)
			go func(kn, source *kademlia.KNode, targetId kademlia.KadId,
				proxySettings proxies.ProxySettings, discovered, probed, failed *sync.Map) {

				defer wg.Done()
				rn := NewNodeClient(kn.Address, proxySettings)
				newNodes, err := rn.FindNode(targetId, source) //FIXME: needs to be changed in FindNode(id *kademlia.KadId) []*kademlia.KNode
				probed.Store(kn.Id, kn)
				if err != nil {
					failed.Store(kn.Id, kn)
					return
				} else {
					for id, addr := range newNodes {
						discovered.LoadOrStore(id, kademlia.NewKNode(id, addr))
					}
				}

			}(ikn, ln.ktable.SelfNode(), targetNode.Id, ln.proxySettings, &discovered, &probed, &failed)
		}

		// Usage of wait group to speed up the process
		wg.Wait()

		/* Insert Nodes into the ktable */
		ln.tsLock.Lock()
		for _, ikn := range alphaClosest {
			_, present := failed.Load(ikn.Id)
			if present {
				ln.ktable.RemoveNode(ikn)
			} else {
				ln.ktable.PushNode(ikn)
			}
		}

		/* resets the alpha closest */
		alphaClosest = make([]*kademlia.KNode, 0)

		/* insert the new nodes and populates the not-yet-probed nodes list*/
		discovered.Range(func(key, value any) bool {
			kn, ok := value.(*kademlia.KNode)
			if !ok {
				return true // continue
			}

			_, present := probed.Load(key)
			if !present { // if not present in failed means that it has not already been probed
				nDiscovered++
				//inserts the new node
				ln.ktable.PushNode(kn)
				alphaClosest = append(alphaClosest, kn)
			}
			return true
		})
		ln.tsLock.Unlock()

		if util.CurrentUnixTime()-startTime >= LOOKUP_ROUND_TIME_THR {
			break
		}

		/* The algo finishes when all the nodes are probed */
		if len(alphaClosest) == 0 {
			break
		}

		kademlia.SortNodesByDistance(targetNode.Id, alphaClosest)

		newClosest := alphaClosest[0]
		if closest != nil {
			previousDistance := closest.Id.EvalDistance(targetNode.Id)
			newDistance := newClosest.Id.EvalDistance(targetNode.Id)
			if newDistance.LessThan(previousDistance) {
				closest = newClosest

				/*
					if the closest node found by this round of find_nodes is actually closest than the previous
					closest node only the alpha nodes are going to be probed, if the round did not found a closer
					node than the previous one then the next round of find_nodes will ask all the unprobed nodes found.
				*/
				if len(alphaClosest) >= kademlia.ALPHA {
					alphaClosest = alphaClosest[:kademlia.ALPHA-1]
				}
			}
		} else {
			closest = newClosest
		}
	}

	logging.LogTrace("Discovered", nDiscovered, "closest nodes to", targetNode.Id)
	return nDiscovered
}

func (ln *LocalNode) StartNodesLookupTask() {
	delay := time.Duration(INITIAL_DELAY)
	ticker := time.NewTicker(delay * time.Second) // node refresh every 15 minutes
	go func() {
		for range ticker.C {
			toLook := make([]*kademlia.KNode, 0)
			startTime := util.CurrentUnixTime()
			if ln.ktable.IsFull() {
				delay = NORMAL_NODES_LOOKUP_DELAY

				for i := 0; i < kademlia.KAD_ID_LEN; i++ {
					if ln.ktable.GetKBuckets()[i].First().LastSeen-uint64(startTime) > uint64(util.TIME_HOUR_UNIX) {
						toLook = append(toLook, ln.ktable.GetKBuckets()[i].GetNodes()[rand.Intn(20)])
					}
				}
			} else {
				delay = NON_FULL_NODES_LOOKUP_DELAY
				for i := 0; i < kademlia.KAD_ID_LEN; i++ {
					// node of distance i from self
					toLook = append(toLook, kademlia.NewKNode(kademlia.GenKadIdFarNBitsFrom(ln.SelfNode().Id, i), ""))
				}
			}
			logging.LogInfo("Started node lookup")
			d := 0
			for i := 0; i < len(toLook); i++ {
				if util.CurrentUnixTime()-startTime >= int64(LOOKUP_ROUND_TIME_THR*len(toLook)) {
					logging.LogWarn("Node lookup taking too much, giving up...")
					break
				}
				d += ln.DoNodesLookup(toLook[i])
			}
			logging.LogInfo("Discovered", d, "new nodes")
			ticker.Reset(delay * time.Second)
		}
	}()
}

func (ln *LocalNode) Shutdown() {
	ln.searchDB.Flush()
	ln.ktable.Flush()
	ln.searchDB.Close()
}
