package core

import (
	"errors"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sniffdogsniff/logging"
	"github.com/sniffdogsniff/proxies"
	"github.com/sniffdogsniff/util"
)

const FIRST_SYNC_LOCK_FILE_NAME = "firstsync.lock"

const NORMAL_SYNC_DELAY = 30000 // milliseconds
const FIRST_SYNC_DELAY = 1      // milliseconds

/* Maximum lenght of searches that a peer can send per time */
const MAX_SYNC_SIZE = 104857600 / SEARCH_RESULT_BYTE_SIZE // 100 MBytes / 512 bytes

type NodeInterface interface {
	Ping(peer Peer) error
	GetStatus() (uint64, uint64)
	GetResultsForSync(timestamp uint64) []SearchResult
	GetMetadataForSync(ts uint64) []ResultMeta
	GetMetadataOf(hashes []Hash256) []ResultMeta
	GetPeersForSync() []Peer
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
	peerDB                PeerDB
	SelfPeer              Peer
}

func NewNode(configs SdsConfig) *LocalNode {
	ln := LocalNode{}
	ln.searchDB.Open(configs.WorkDirPath, configs.searchDBMaxCacheSize)
	ln.peerDB.Open(configs.WorkDirPath, configs.KnownPeers)
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

func (ln *LocalNode) SetNodeAddress(addr string) {
	ln.SelfPeer = NewPeer(addr)
}

func (ln *LocalNode) firstSyncLockFileExists() bool {
	return util.FileExists(ln.firstSyncLockFilePath)
}

func (ln *LocalNode) CalculateAgreementThreshold() int {
	return 2 * int(math.Log10(float64(ln.peerDB.Count())))
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

func (ln *LocalNode) GetPeersForSync() []Peer {
	ln.tsLock.Lock()
	results := ln.peerDB.GetAll()
	ln.tsLock.Unlock()
	return results
}

func (ln *LocalNode) Ping(peer Peer) error {
	if ln.firstSyncLockFileExists() {
		return errors.New("first sync, handshake refused")
	}
	ln.tsLock.Lock()
	ln.peerDB.SyncFrom([]Peer{peer}, ln.SelfPeer)
	ln.tsLock.Unlock()
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

func (ln *LocalNode) SyncWithPeer() {
	firstSyncFileExists := ln.firstSyncLockFileExists()

	ln.tsLock.Lock()
	p := ln.peerDB.GetRandomPeer()
	rn := NewNodeClient(p, ln.proxySettings)
	ln.tsLock.Unlock()
	logging.LogInfo("Sync with ", p.Address)

	err, _ := rn.Ping(ln.SelfPeer)
	/* if the first peer does not respond and the db is
	empty first sync flag is set back to false to avoid
	infinite loop cycle blocking the all node.*/

	if firstSyncFileExists {
		ln.FirstSync = err == nil
	}
	if err != nil {
		logging.LogWarn("Unsuccessful peer handshake: aborting sync: caused by", err)
		return
	}

	/* Sync of peers */
	newPeers, _ := rn.GetPeersForSync()
	logging.LogTrace("Received", len(newPeers), "peers")

	if len(newPeers) > 0 {
		ln.tsLock.Lock()
		ln.peerDB.SyncFrom(newPeers, ln.SelfPeer)
		ln.tsLock.Unlock()
	}

	searchesTimestamp, metasTimestamp := ln.GetStatus()
	remoteSearchesTimestamp, remoteMetasTimestamp, _ := rn.GetStatus()
	logging.LogTrace("Remote Time", remoteSearchesTimestamp)

	if firstSyncFileExists {
		if searchesTimestamp >= remoteSearchesTimestamp && metasTimestamp >= remoteMetasTimestamp {
			os.Remove(ln.firstSyncLockFilePath)
		}
	}

	/* Sync of searches */
	if searchesTimestamp < remoteSearchesTimestamp {
		newSearches, _ := rn.GetResultsForSync(searchesTimestamp)
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
		newMetadata, _ := rn.GetMetadataForSync(metasTimestamp)
		logging.LogTrace("Received", len(newMetadata), "results metadata")

		if (!firstSyncFileExists) && ln.canInvalidate && len(newMetadata) > 0 {
			hashList := make([]Hash256, 0)
			invalidations := make(map[Hash256]int8, 0)
			for _, rm := range newMetadata {
				if rm.Invalidated > INVALIDATION_LEVEL_NONE {
					invalidations[rm.ResultHash] = 1
					hashList = append(hashList, rm.ResultHash)
				}
			}

			nConfirmations := ln.CalculateAgreementThreshold()
			for _, pi := range ln.peerDB.GetRandomPeerList(nConfirmations) {
				rni := NewNodeClient(pi, ln.proxySettings)
				rmiMetas, _ := rni.GetMetadataOf(hashList)
				for _, rmi := range rmiMetas {
					if rmi.Invalidated > INVALIDATION_LEVEL_NONE {
						invalidations[rmi.ResultHash] += 1
					}
				}
			}

			for _, rm := range newMetadata {
				rm.Invalidated = invalidations[rm.ResultHash]
				if rm.Invalidated >= int8(nConfirmations) {
					rm.Invalidated = INVALIDATION_LEVEL_INVALIDATED
				}
			}
		}

		if len(newMetadata) > 0 {
			ln.tsLock.Lock()
			ln.searchDB.SyncResultsMetadataFrom(newMetadata)
			ln.tsLock.Unlock()
		}
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
				ln.peerDB.Flush()
				ln.tsLock.Unlock()
				syncCycles = 0
			}
		}
	}()
}

func (ln *LocalNode) Shutdown() {
	ln.searchDB.Flush()
	ln.peerDB.Flush()
	ln.searchDB.Close()
}
