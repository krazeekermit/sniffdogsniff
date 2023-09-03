package core

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sniffdogsniff/kademlia"
	"github.com/sniffdogsniff/logging"
	"github.com/sniffdogsniff/proxies"
	"github.com/sniffdogsniff/util"
)

const FIRST_SYNC_LOCK_FILE_NAME = "firstsync.lock"

const NORMAL_SYNC_DELAY = 30000 // milliseconds
const FIRST_SYNC_DELAY = 1      // milliseconds

const SELF_PEER_FILE_NAME = "selfnode.dat"

/* Maximum lenght of searches that a peer can send per time */
const MAX_SYNC_SIZE = 104857600 / SEARCH_RESULT_BYTE_SIZE // 100 MBytes / 512 bytes

var CANT_SYNC_ERROR = errors.New("first sync, cant sync from this")

type NodeInterface interface {
	Ping(id kademlia.KadId, address string) error
	GetStatus() (uint64, uint64)
	GetResultsForSync(timestamp uint64) []SearchResult
	GetMetadataForSync(ts uint64) []ResultMeta
	GetMetadataOf(hashes []Hash256) []ResultMeta
	GetKClosestNodes() map[kademlia.KadId]string
}

type LocalNode struct {
	proxySettings         proxies.ProxySettings
	canInvalidate         bool
	tsLock                *sync.Mutex // tread safe access from different threads, the NodeServer the WebUi, the SyncWithPeers()
	searchDB              SearchDB
	searchEngines         map[string]SearchEngine
	minResultsThr         int
	FirstSync             bool
	firstSyncLockFilePath string
	selfNodeFilePath      string
	ktable                *kademlia.KadRoutingTable
	SelfNode              *kademlia.KNode
}

func NewNode(configs SdsConfig) *LocalNode {
	ln := LocalNode{}
	ln.selfNodeFilePath = filepath.Join(configs.WorkDirPath, SELF_PEER_FILE_NAME)
	ln.ktable = kademlia.NewKadRoutingTable(ln.GetSelfNode())
	ln.ktable.Open(configs.WorkDirPath)
	if ln.ktable.IsEmpty() {
		for id, addr := range configs.KnownPeers {
			logging.LogTrace(addr)
			ln.ktable.PushNode(kademlia.NewKNode(id, addr))
		}
	}

	ln.searchDB.Open(configs.WorkDirPath, configs.searchDBMaxCacheSize)
	ln.proxySettings = configs.proxySettings
	ln.canInvalidate = configs.AllowResultsInvalidation
	ln.tsLock = &sync.Mutex{}
	ln.searchEngines = configs.searchEngines
	ln.minResultsThr = 10 // 10 placeholder number will be defined in SdsConfigs
	ln.firstSyncLockFilePath = filepath.Join(configs.WorkDirPath, FIRST_SYNC_LOCK_FILE_NAME)
	if ln.searchDB.IsEmpty() {
		os.Create(ln.firstSyncLockFilePath)
	}
	if util.FileExists(ln.firstSyncLockFilePath) {
		ln.FirstSync = true
		logging.LogWarn("Your Node is not already synced with the rest of the network, you are in FIRSTSYNC mode!")
	}
	return &ln
}

func (ln *LocalNode) GetSelfNode() *kademlia.KNode {
	if util.FileExists(ln.selfNodeFilePath) {
		fp, err := os.OpenFile(ln.selfNodeFilePath, os.O_RDONLY, 0600)
		if err != nil {
			logging.LogError("failed to open file", ln.selfNodeFilePath, err)
			goto fail_read
		}
		kadIdBytez := make([]byte, 20)
		n, err := fp.Read(kadIdBytez)
		if err != nil || n != 20 {
			fp.Close()
			goto fail_read
		}
		kadId := kademlia.KadIdFromBytes(kadIdBytez)

		addressBytez := make([]byte, 0)
		for {
			buf := make([]byte, 1024)
			n, err := fp.Read(buf)
			if err != nil {
				fp.Close()
				goto fail_read
			}
			addressBytez = append(addressBytez, buf[:n]...)
			if n < 1024 {
				break
			}
		}
		fp.Close()
		addr := string(addressBytez)

		logging.LogInfo("Using cahched node address", addr, "with id", kadId)
		return &kademlia.KNode{
			Id:      kadId,
			Address: addr,
		}
	}
fail_read:
	return kademlia.NewKNode(kademlia.NewRandKadId(), "")
}

func (ln *LocalNode) SetNodeAddress(addr string) {
	selfNode := kademlia.NewKNode(kademlia.NewKadId(addr), addr)
	ln.ktable.SetSelfNode(selfNode)
	ln.SelfNode = selfNode
	fp, err := os.OpenFile(ln.selfNodeFilePath, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		logging.LogError("failed to open file", ln.selfNodeFilePath, err)
		return
	}
	fp.Write(selfNode.Id.ToBytes())
	fp.Write([]byte(selfNode.Address))
	fp.Close()
}

func (ln *LocalNode) firstSyncLockFileExists() bool {
	return util.FileExists(ln.firstSyncLockFilePath)
}

func (ln *LocalNode) CalculateAgreementThreshold() int {
	return 2
}

func (ln *LocalNode) GetMetadataOf(hashes []Hash256) []ResultMeta {
	ln.tsLock.Lock()
	metadata := ln.searchDB.GetMetadataOf(hashes)
	ln.tsLock.Unlock()
	return metadata
}

func (ln *LocalNode) GetMetadataForSync(ts uint64) []ResultMeta {
	ln.tsLock.Lock()
	metadata := ln.searchDB.GetMetadataForSync(ts, MAX_SYNC_SIZE)
	ln.tsLock.Unlock()
	return metadata
}

func (ln *LocalNode) GetResultsForSync(timestamp uint64) []SearchResult {
	ln.tsLock.Lock()
	results := ln.searchDB.GetForSync(timestamp, MAX_SYNC_SIZE)
	ln.tsLock.Unlock()
	return results
}

func (ln *LocalNode) GetKClosestNodes() map[kademlia.KadId]string {
	ln.tsLock.Lock()
	closest := ln.ktable.GetKClosest()
	closestMap := make(map[kademlia.KadId]string)
	for _, ikn := range closest {
		closestMap[ikn.Id] = ikn.Address
	}
	ln.tsLock.Unlock()
	return closestMap
}

func (ln *LocalNode) Ping(id kademlia.KadId, addr string) error {
	ln.tsLock.Lock()
	ln.ktable.PushNode(kademlia.NewKNode(id, addr))
	ln.tsLock.Unlock()

	if ln.firstSyncLockFileExists() {
		return CANT_SYNC_ERROR
	}
	return nil
}

func (ln *LocalNode) GetStatus() (uint64, uint64) {
	return ln.searchDB.LastTimestamp, ln.searchDB.LastMetaTimestamp
}

func (ln *LocalNode) InsertSearchResult(sr SearchResult) {
	ln.tsLock.Lock()
	ln.searchDB.InsertResult(sr)
	ln.tsLock.Unlock()
}

func (ln *LocalNode) InvalidateSearchResult(h Hash256) {
	ln.tsLock.Lock()
	rm, err := ln.searchDB.GetMetaByHash(h)
	if err != nil {
		ln.searchDB.SetInvalidationLevel(h, rm.Invalidated+1)
	}
	ln.tsLock.Unlock()
}

func (ln *LocalNode) UpdateResultScore(hash string) {
	ln.tsLock.Lock()
	ln.searchDB.UpdateResultScore(B64UrlsafeStringToHash(hash), 1)
	ln.tsLock.Unlock()
}

func (ln *LocalNode) DoSearch(query string) []SearchResult {
	results := make([]SearchResult, 0)

	ln.tsLock.Lock() // tread safe access to SearchDB
	results = append(results, ln.searchDB.DoSearch(query)...)
	ln.tsLock.Unlock()

	if len(results) > ln.minResultsThr {
		return results
	}

	results = append(results, DoParallelSearchOnExtEngines(ln.searchEngines, query)...)

	ln.searchDB.SyncFrom(results)
	return results
}

/*
 * SniffDogSniff uses:
 *  * Kademlia for peers discovery;
 *  * Epidemic Gossip protocol SI model pull method for syncing SearchResults;
 */

func (ln *LocalNode) SyncWithPeer() {
	firstSyncFileExists := ln.firstSyncLockFileExists()

	ln.tsLock.Lock()
	closestNodes := ln.ktable.GetKClosest()
	ln.tsLock.Unlock()

	for _, ikn := range closestNodes {
		var searchesTimestamp, metasTimestamp, remoteSearchesTimestamp, remoteMetasTimestamp uint64

		rn := NewNodeClient(ikn.Address, ln.proxySettings)
		logging.LogInfo("Sync with ", ikn.Address)

		remoteErr, err := rn.Ping(ln.SelfNode.Id, ln.SelfNode.Address)
		/* if the first peer does not respond and the db is
		empty first sync flag is set back to false to avoid
		infinite loop cycle blocking the all node.*/

		if firstSyncFileExists {
			ln.FirstSync = err == nil
		}
		if err != nil {
			logging.LogWarn("Unsuccessful peer ping: aborting sync: caused by", err)
			goto sync_fail
		}

		/* Sync of peers */
		if !ln.ktable.IsFull() {
			newNodes, err := rn.GetKClosestNodes()
			if err != nil {
				goto sync_fail
			}

			logging.LogTrace("Received", len(newNodes), "new nodes")
			if len(newNodes) > 0 {
				ln.tsLock.Lock()
				for id, addr := range newNodes {
					if id.Eq(ln.SelfNode.Id) {
						continue
					}
					ln.ktable.PushNode(kademlia.NewKNode(id, addr))
				}
				ln.tsLock.Unlock()
			}
		}

		if remoteErr == CANT_SYNC_ERROR {
			continue
		}

		searchesTimestamp, metasTimestamp = ln.GetStatus()
		remoteSearchesTimestamp, remoteMetasTimestamp, err = rn.GetStatus()
		if err != nil {
			goto sync_fail
		}

		logging.LogTrace("Remote Time", remoteSearchesTimestamp)
		if firstSyncFileExists {
			if searchesTimestamp >= remoteSearchesTimestamp && metasTimestamp >= remoteMetasTimestamp {
				os.Remove(ln.firstSyncLockFilePath)
			}
		}

		/* Sync of searches */
		if searchesTimestamp < remoteSearchesTimestamp {
			newSearches, err := rn.GetResultsForSync(searchesTimestamp)
			if err != nil {
				ln.tsLock.Lock()
				ln.ktable.RemoveNode(ikn)
				ln.tsLock.Unlock()
				continue
			}

			logging.LogTrace("Received", len(newSearches), "searches")
			if len(newSearches) > 0 {
				ln.tsLock.Lock()
				ln.searchDB.SyncFrom(newSearches)
				ln.tsLock.Unlock()
			}
		} else {
			if ln.FirstSync {
				ln.FirstSync = false
			}
		}

		/* Sync of metadatada of searches */
		if metasTimestamp < remoteMetasTimestamp {
			newMetadata, err := rn.GetMetadataForSync(metasTimestamp)
			if err != nil {
				goto sync_fail
			}
			logging.LogTrace("Received", len(newMetadata), "results metadata")

			// the invalidation sync is disabled until the kademlia transition is fully completed
			// if (!firstSyncFileExists) && ln.canInvalidate && len(newMetadata) > 0 {
			// 	hashList := make([]Hash256, 0)
			// 	invalidations := make(map[Hash256]int8, 0)
			// 	for _, rm := range newMetadata {
			// 		if rm.Invalidated > INVALIDATION_LEVEL_NONE {
			// 			invalidations[rm.ResultHash] = 1
			// 			hashList = append(hashList, rm.ResultHash)
			// 		}
			// 	}
			// }
			// 		nConfirmations := ln.CalculateAgreementThreshold()
			// 		for _, pi := range ln.peerDB.GetRandomPeerList(nConfirmations) {
			// 			rni := NewNodeClient(pi, ln.proxySettings)
			// 			rmiMetas, _ := rni.GetMetadataOf(hashList)
			// 			for _, rmi := range rmiMetas {
			// 				if rmi.Invalidated > INVALIDATION_LEVEL_NONE {
			// 					invalidations[rmi.ResultHash] += 1
			// 				}
			// 			}
			// 		}

			// 		for _, rm := range newMetadata {
			// 			rm.Invalidated = invalidations[rm.ResultHash]
			// 			if rm.Invalidated >= int8(nConfirmations) {
			// 				rm.Invalidated = INVALIDATION_LEVEL_INVALIDATED
			// 			}
			// 		}

			if len(newMetadata) > 0 {
				ln.tsLock.Lock()
				ln.searchDB.SyncResultsMetadataFrom(newMetadata)
				ln.tsLock.Unlock()
			}
		}
		continue

	sync_fail:
		ln.tsLock.Lock()
		ln.ktable.RemoveNode(ikn)
		ln.tsLock.Unlock()
	}

}

func (ln *LocalNode) StartSyncTask() {
	ticker := time.NewTicker(time.Duration(FIRST_SYNC_DELAY) * time.Millisecond)
	syncCycles := 0
	go func() {
		for range ticker.C {
			delay := NORMAL_SYNC_DELAY
			if ln.FirstSync {
				delay = FIRST_SYNC_DELAY
			}
			ticker.Reset(time.Duration(delay) * time.Millisecond)
			ln.SyncWithPeer()
			syncCycles++
			if syncCycles >= 5 { // every 5 cycles the data is flushed to disk (number choice is totally hempiric)
				ln.tsLock.Lock()
				ln.searchDB.Flush()
				ln.ktable.Flush()
				ln.tsLock.Unlock()
				syncCycles = 0
			}
		}
	}()
}

func (ln *LocalNode) Shutdown() {
	ln.searchDB.Flush()
	ln.ktable.Flush()
	ln.searchDB.Close()
}
