package sds

import (
	"sync"
	"time"

	"gitlab.com/sniffdogsniff/util"
	"gitlab.com/sniffdogsniff/util/logging"
)

type LocalNode struct {
	proxySettings ProxySettings
	tsLock        sync.Mutex // tread safe access from different threads, the NodeServer the WebUi, the SyncWithPeers()
	searchDB      SearchDB
	searchEngines map[string]SearchEngine
	minResultsThr int
	peerDB        PeerDB
	SelfPeer      Peer
}

func GetNodeInstance(configs SdsConfig) *LocalNode {
	ln := LocalNode{}
	ln.searchDB.Open(configs.searchDatabasePath, configs.searchDBMaxCacheSize)
	ln.peerDB.Open(configs.peersDatabasePath, configs.KnownPeers)
	ln.proxySettings = configs.proxySettings
	ln.tsLock = sync.Mutex{}
	ln.searchEngines = configs.searchEngines
	ln.minResultsThr = 10 // 10 placeholder number will be defined in SdsConfigs
	ln.SelfPeer = configs.ServiceSettings.PeerInfo
	return &ln
}

func (ln *LocalNode) GetMetadataOf(hashes [][32]byte) []ResultMeta {
	ln.tsLock.Lock()
	metadata := ln.searchDB.GetMetadataOf(hashes)
	ln.tsLock.Unlock()
	return metadata
}

func (ln *LocalNode) GetMetadataForSync(ts uint64) []ResultMeta {
	ln.tsLock.Lock()
	metadata := ln.searchDB.GetMetadataForSync(ts)
	ln.tsLock.Unlock()
	return metadata
}

func (ln *LocalNode) GetResultsForSync(timestamp uint64) []SearchResult {
	ln.tsLock.Lock()
	results := ln.searchDB.GetForSync(timestamp)
	ln.tsLock.Unlock()
	return results
}

func (ln *LocalNode) getPeersForSync() []Peer {
	ln.tsLock.Lock()
	results := ln.peerDB.GetAll()
	ln.tsLock.Unlock()
	return results
}

func (ln *LocalNode) Handshake(peer Peer) {
	ln.tsLock.Lock()
	ln.peerDB.SyncFrom([]Peer{peer})
	ln.tsLock.Unlock()
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
	ln.tsLock.Lock()
	allPeers := ln.peerDB.GetAll()
	p := ln.peerDB.GetRandomPeer()
	ln.tsLock.Unlock()
	logging.LogInfo("Sync with ", p.Address)

	err := p.Handshake(ln.proxySettings, ln.SelfPeer)
	if err != nil {
		logging.LogWarn("Unsuccessful peer handshake: aborting sync")
		return
	}

	searchesTimestamp := ln.searchDB.LastTimestamp
	metasTimestamp := ln.searchDB.LastMetaTimestamp

	newSearches := p.GetResultsForSync(ln.proxySettings, searchesTimestamp)
	logging.LogTrace("Received", len(newSearches), "searches")
	newMetadata := p.GetMetadataForSync(ln.proxySettings, metasTimestamp)
	logging.LogTrace("Received", len(newMetadata), "results metadata")
	newPeers := p.GetPeersForSync(ln.proxySettings)
	logging.LogTrace("Received", len(newPeers), "peers")

	if true { // invalidation link feature can be disabled by the user
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

		ln.tsLock.Lock()
		ln.searchDB.SyncFrom(newSearches)
		ln.searchDB.SyncResultsMetadataFrom(newMetadata)
		ln.peerDB.SyncFrom(newPeers)
		ln.tsLock.Unlock()
	}
}

func (ln *LocalNode) StartSyncTask() {
	ticker := time.NewTicker(30 * time.Second)
	syncCycles := 0
	go func() {
		for range ticker.C {
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
}
