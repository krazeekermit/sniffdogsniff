package core

import (
	"errors"
	"math/rand"
	"path/filepath"
	"sync"
	"time"

	"github.com/sniffdogsniff/kademlia"
	"github.com/sniffdogsniff/logging"
	"github.com/sniffdogsniff/util"
)

const NODE = "node"
const NODES_LOOKUP = "lookup"

const FIRST_SYNC_LOCK_FILE_NAME = "firstsync.lock"

const LOOKUP_ROUND_TIMEOUT = 20         // seconds
const LOOKUP_TIMEOUT = 60               // seconds
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
	CheckNode(id kademlia.KadId, addr string) bool
}

type LocalNode struct {
	canInvalidate    bool
	tsLock           *sync.Mutex // tread safe access from different threads, the NodeServer the WebUi, the SyncWithPeers()
	searchDB         SearchDB
	Crawler          *Crawler
	minResultsThr    int
	selfNodeFilePath string
	ktable           *kademlia.KadRoutingTable
	knownNodes       map[kademlia.KadId]string
	nodesBlacklist   map[kademlia.KadId]string
}

func NewLocalNode(configs SdsConfig) *LocalNode {
	ln := &LocalNode{}
	ln.knownNodes = configs.KnownPeers
	ln.nodesBlacklist = configs.PeersBlacklist
	ln.selfNodeFilePath = filepath.Join(configs.WorkDirPath, SELF_PEER_FILE_NAME)
	ln.ktable = kademlia.NewKadRoutingTable()
	ln.ktable.Open(configs.WorkDirPath)

	ln.searchDB.Open(configs.WorkDirPath, configs.searchDBMaxCacheSize, 3600*24)
	ln.canInvalidate = configs.AllowResultsInvalidation
	ln.tsLock = &sync.Mutex{}
	ln.Crawler = NewCrawler(configs.searchEngines)
	ln.Crawler.SetUpdateCallback(ln.InsertSearchResult)
	ln.minResultsThr = 10 // 10 placeholder number will be defined in SdsConfigs
	return ln
}

func (ln *LocalNode) SelfNode() *kademlia.KNode {
	return ln.ktable.SelfNode()
}

func (ln *LocalNode) SetNodeAddress(addr string) {
	ln.tsLock.Lock()
	ln.ktable.SetSelfNode(kademlia.NewKNode(kademlia.NewKadIdFromAddrStr(addr), addr))
	if ln.ktable.IsEmpty() {
		for id, addr := range ln.knownNodes {
			logging.Debugf(NODE, "set address: %s", addr)
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

func (ln *LocalNode) CheckNode(id kademlia.KadId, addr string) bool {
	for bid, baddr := range ln.nodesBlacklist {
		if bid.Eq(id) || baddr == addr {
			return false
		}
	}
	return kademlia.NewKadIdFromAddrStr(addr).Eq(id)
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
	failed := kademlia.NewKNodesMap()
	emptyProbed := kademlia.NewKNodesMap()

	resultsLock := sync.Mutex{}
	results := make(map[Hash256]SearchResult)

	wgCount := 0
	for _, ikn := range nodes {
		_, present := failed.Get(ikn.Id)
		if present {
			// if it is a failed node avoid to contact it again
			continue
		}
		// if node is self node not contact it and store results inside of it
		if ikn.Id.Eq(ln.SelfNode().Id) {
			resultsLock.Lock()
			for _, v := range ln.FindResults(query) {
				results[v.ResultHash] = v
			}
			resultsLock.Unlock()
			continue
		}

		wg.Add(1)
		wgCount++
		go func(kn, source *kademlia.KNode, query string, failed, emptyProbed *kademlia.KNodesMap,
			results *map[Hash256]SearchResult, resultsLock *sync.Mutex) {
			defer wg.Done()

			rn := NewNodeClient(kn.Address)
			vals, err := rn.FindResults(query, source)
			if err != nil {
				failed.Put(kn)
				return
			}
			if len(vals) > 0 {
				resultsLock.Lock()
				for _, v := range vals {
					(*results)[v.ResultHash] = v
				}
				resultsLock.Unlock()
			} else {
				emptyProbed.Put(kn)
			}
		}(ikn, ln.ktable.SelfNode(), query, failed, emptyProbed, &results, &resultsLock)

		if wgCount >= kademlia.ALPHA {
			wgCount = 0
			wg.Wait()
		}
	}

	ln.tsLock.Lock()
	for _, k := range failed.Keys() {
		kn, _ := failed.Get(k)
		ln.ktable.RemoveNode(kn)
	}
	ln.tsLock.Unlock()

	rSlice := make([]SearchResult, 0)
	for _, sr := range results {
		rSlice = append(rSlice, sr)
	}

	if len(rSlice) == 0 {
		/*
			If no values were found in any nodes then rely on the centralized external
			search engines. The found results are stored in the network.
		*/
		rSlice = append(rSlice, ln.Crawler.DoSearch(query)...)
		ln.PublishResults(rSlice)
		return rSlice
	}

	// re stores the values on nodes that are supposed to have the value but does not have it
	for _, kid := range emptyProbed.Keys() {
		kn, _ := emptyProbed.Get(kid)
		rn := NewNodeClient(kn.Address)
		for _, sr := range rSlice {
			rn.StoreResult(sr, ln.SelfNode())
		}
	}

	return rSlice
}

/*
	Publish Results
*/

func (ln *LocalNode) PublishResults(results []SearchResult) {
	failed := kademlia.NewKNodesMap()
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
			_, present := failed.Get(ikn.Id)
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
			go func(kn, source *kademlia.KNode, value SearchResult, failed *kademlia.KNodesMap) {
				defer wg.Done()

				rn := NewNodeClient(kn.Address)
				err := rn.StoreResult(sr, source)
				if err != nil {
					failed.Put(kn)
				}
			}(ikn, ln.ktable.SelfNode(), sr, failed)

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

	for _, kid := range failed.Keys() {
		kn, _ := failed.Get(kid)
		ln.ktable.RemoveNode(kn)
	}
	ln.tsLock.Unlock()
}

func (ln *LocalNode) StartPublishTask() {
	ticker := time.NewTicker(INITIAL_DELAY * time.Second)
	go func() {
		for range ticker.C {
			ticker.Reset(time.Duration(util.TIME_HOUR_UNIX) * time.Second)
			ln.tsLock.Lock()
			results := ln.searchDB.ResultsToPublish()
			logging.Infof(NODE, "Publishing %d results...", len(results))
			ln.tsLock.Unlock()
			ln.PublishResults(results)
		}
	}()
}

/*
	Node Lookup
*/

func (ln *LocalNode) DoNodesLookup(targetNode *kademlia.KNode, checkNode bool) int {
	ln.tsLock.Lock()
	alphaClosest := ln.ktable.GetNClosestTo(targetNode.Id, kademlia.ALPHA)
	ln.tsLock.Unlock()

	if len(alphaClosest) == 0 {
		return 0
	}

	nDiscovered := 0
	startTime := util.CurrentUnixTime()

	var wg sync.WaitGroup
	discovered := kademlia.NewKNodesMap()
	probed := kademlia.NewKNodesMap()
	failed := kademlia.NewKNodesMap()

	self := ln.SelfNode()
	probed.Put(self)

	for {
		for _, ikn := range alphaClosest {
			if ikn.Id.Eq(self.Id) {
				continue
			}

			wg.Add(1)
			go func(kn, source *kademlia.KNode, targetId kademlia.KadId, discovered, probed, failed *kademlia.KNodesMap) {

				defer wg.Done()
				rn := NewNodeClient(kn.Address)
				newNodes, err := rn.FindNode(targetId, source)
				probed.Put(kn)
				if err != nil {
					logging.Errorf(NODES_LOOKUP, err.Error())
					failed.Put(kn)
					return
				} else {
					for id, addr := range newNodes {
						discovered.Put(kademlia.NewKNode(id, addr))
					}
				}

			}(ikn, ln.ktable.SelfNode(), targetNode.Id, discovered, probed, failed)
		}

		// Usage of wait group to speed up the process
		wg.Wait()

		/* Insert Nodes into the ktable */
		ln.tsLock.Lock()
		for _, ikn := range alphaClosest {
			_, present := failed.Get(ikn.Id)
			if present {
				ln.ktable.RemoveNode(ikn)
			} else {
				ln.ktable.PushNode(ikn)
			}
		}

		/* resets the alpha closest */
		alphaClosest = make([]*kademlia.KNode, 0)

		/* insert the new nodes and populates the not-yet-probed nodes list*/
		for _, k := range discovered.Keys() {
			kn, _ := discovered.Get(k)

			if checkNode {
				if !ln.CheckNode(kn.Id, kn.Address) {
					continue
				}
			}

			_, present := probed.Get(k)
			if !present { // if not present in failed means that it has not already been probed
				nDiscovered++
				//inserts the new node
				ln.ktable.PushNode(kn)
				alphaClosest = append(alphaClosest, kn)
			}
		}
		ln.tsLock.Unlock()

		if util.CurrentUnixTime()-startTime >= LOOKUP_ROUND_TIMEOUT {
			break
		}

		/* The algo finishes when all the nodes are probed */
		if len(alphaClosest) == 0 {
			break
		}

		kademlia.SortNodesByDistance(targetNode.Id, alphaClosest)
		if len(alphaClosest) >= kademlia.ALPHA {
			alphaClosest = alphaClosest[:kademlia.ALPHA]
		}
	}

	logging.Debugf(NODES_LOOKUP, "Discovered %d closest nodes to %s", nDiscovered, targetNode.Id)
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
			logging.Infof(NODES_LOOKUP, "Started node lookup")
			d := 0
			for i := 0; i < len(toLook); i++ {
				if util.CurrentUnixTime()-startTime >= int64(LOOKUP_TIMEOUT) {
					logging.Warnf(NODES_LOOKUP, "Nodes lookup taking too much, giving up...")
					break
				}
				d += ln.DoNodesLookup(toLook[i], true)
			}
			logging.Infof(NODES_LOOKUP, "Discovered %d new nodes", d)
			ticker.Reset(delay * time.Second)
		}
	}()
}

func (ln *LocalNode) Shutdown() {
	ln.searchDB.Flush()
	ln.ktable.Flush()
	ln.searchDB.Close()
}
