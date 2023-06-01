package sds

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sniffdogsniff/util"
	"github.com/sniffdogsniff/util/logging"
)

const FIRST_SYNC_LOCK_FILE_NAME = "firstsync.lock"

const NORMAL_SYNC_DELAY = 30000 // milliseconds
const FIRST_SYNC_DELAY = 1      // milliseconds

/* Maximum lenght of searches that a peer can send per time */
const MAX_SYNC_SIZE = 104857600 / SEARCH_RESULT_BYTE_SIZE // 100 MBytes / 512 bytes

type LocalNode struct {
	proxySettings         ProxySettings
	tsLock                sync.Mutex // tread safe access from different threads, the NodeServer the WebUi, the SyncWithPeers()
	searchDB              SearchDB
	searchEngines         map[string]SearchEngine
	minResultsThr         int
	FirstSync             bool
	firstSyncLockFilePath string
	peerDB                PeerDB
	SelfPeer              Peer
}

func GetNodeInstance(configs SdsConfig) *LocalNode {
	ln := LocalNode{}
	ln.searchDB.Open(configs.workDirPath, configs.searchDBMaxCacheSize)
	ln.peerDB.Open(configs.workDirPath, configs.KnownPeers)
	ln.proxySettings = configs.proxySettings
	ln.tsLock = sync.Mutex{}
	ln.searchEngines = configs.searchEngines
	ln.minResultsThr = 10 // 10 placeholder number will be defined in SdsConfigs
	ln.firstSyncLockFilePath = filepath.Join(configs.workDirPath, FIRST_SYNC_LOCK_FILE_NAME)
	if ln.searchDB.IsEmpty() {
		os.Create(ln.firstSyncLockFilePath)
	}
	if util.FileExists(ln.firstSyncLockFilePath) {
		ln.FirstSync = true
		logging.LogWarn("Your Node is not already synced with the rest of the network, you are in FIRSTSYNC mode!")
	}
	ln.SelfPeer = configs.ServiceSettings.PeerInfo
	return &ln
}

func (ln *LocalNode) firstSyncLockFileExists() bool {
	return util.FileExists(ln.firstSyncLockFilePath)
}

func (ln *LocalNode) GetMetadataOf(hashes [][32]byte) []ResultMeta {
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

func (ln *LocalNode) getPeersForSync() []Peer {
	ln.tsLock.Lock()
	results := ln.peerDB.GetAll()
	ln.tsLock.Unlock()
	return results
}

func (ln *LocalNode) Handshake(peer Peer) error {
	if ln.firstSyncLockFileExists() {
		return errors.New("First sync, handshake refused")
	}
	ln.tsLock.Lock()
	ln.peerDB.SyncFrom([]Peer{peer})
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

func (ln *LocalNode) InvalidateSearchResult(h [32]byte) {
	ln.tsLock.Lock()
	ln.searchDB.InvalidateResult(h)
	ln.tsLock.Unlock()
}

func (ln *LocalNode) UpdateResultScore(hash string) {
	ln.tsLock.Lock()
	ln.searchDB.UpdateResultScore(util.B64UrlsafeStringToHash(hash), 1)
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

	for _, engine := range ln.searchEngines {
		results = append(results, engine.DoSearch(query)...)
	}
	ln.searchDB.SyncFrom(results)
	return results
}

func (ln *LocalNode) SyncWithPeer() {
	firstSyncFileExists := ln.firstSyncLockFileExists()

	ln.tsLock.Lock()
	allPeers := ln.peerDB.GetAll()
	p := ln.peerDB.GetRandomPeer()
	ln.tsLock.Unlock()
	logging.LogInfo("Sync with ", p.Address)

	err := p.Handshake(ln.proxySettings, ln.SelfPeer)
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
	newPeers := p.GetPeersForSync(ln.proxySettings)
	logging.LogTrace("Received", len(newPeers), "peers")

	if len(newPeers) > 0 {
		ln.tsLock.Lock()
		ln.peerDB.SyncFrom(newPeers)
		ln.tsLock.Unlock()
	}

	searchesTimestamp, metasTimestamp := ln.GetStatus()
	remoteSearchesTimestamp, remoteMetasTimestamp := p.GetStatus(ln.proxySettings)

	if firstSyncFileExists {
		if searchesTimestamp >= remoteSearchesTimestamp && metasTimestamp >= remoteMetasTimestamp {
			os.Remove(ln.firstSyncLockFilePath)
		}
	}

	/* Sync of searches */
	if searchesTimestamp < remoteSearchesTimestamp {
		newSearches := p.GetResultsForSync(ln.proxySettings, searchesTimestamp)
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
		newMetadata := p.GetMetadataForSync(ln.proxySettings, metasTimestamp)
		logging.LogTrace("Received", len(newMetadata), "results metadata")

		if len(newMetadata) > 0 {
			invalidationTable := make(map[[32]byte]int, 0)
			invalidationTableKeyList := util.MapKeys(invalidationTable)
			newMetadataPtrMap := make(map[[32]byte]*ResultMeta)

			for _, m := range newMetadata {
				newMetadataPtrMap[m.ResultHash] = &m
				if m.Invalidated != NONE {
					invalidationTable[m.ResultHash] = 0
				}
			}
			for _, po := range allPeers {
				for _, mo := range po.GetMetadataOf(ln.proxySettings, invalidationTableKeyList) {
					if mo.Invalidated == INVALIDATED {
						invalidationTable[mo.ResultHash] += 1
					}
				}
			}
			for rHash, inv := range invalidationTable {
				if inv > (len(allPeers) / 2) {
					if newMetadataPtrMap[rHash].Invalidated < INVALIDATED {
						newMetadataPtrMap[rHash].Invalidated += 1
					} else if newMetadataPtrMap[rHash].Invalidated == INVALIDATED {
						newMetadataPtrMap[rHash].Invalidated = PENDING
					}
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
				ln.tsLock.Unlock()
				syncCycles = 0
			}
		}
	}()
}

func (ln *LocalNode) Shutdown() {
	ln.searchDB.Flush()
	ln.searchDB.Close()
}
