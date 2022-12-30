package sds

import (
	"sync"
	"time"

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

func InitNode(configs SdsConfig) LocalNode {
	ln := LocalNode{}
	ln.searchDB.Open(configs.searchDatabasePath)
	ln.peerDB.Open(configs.peersDatabasePath, configs.KnownPeers)
	ln.proxySettings = configs.proxySettings
	ln.tsLock = sync.Mutex{}
	ln.searchEngines = configs.searchEngines
	ln.minResultsThr = 10 // 10 placeholder number will be defined in SdsConfigs
	ln.SelfPeer = configs.NodePeerInfo
	return ln
}

func (ln *LocalNode) GetResultsForSync(hashes [][32]byte) []SearchResult {
	ln.tsLock.Lock()
	results := ln.searchDB.GetForSync(hashes)
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
	ln.searchDB.InsertRow(sr)
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

func (ln *LocalNode) SyncWithPeers() {
	for {
		time.Sleep(30 * time.Second)

		ln.tsLock.Lock()
		peers := ln.peerDB.GetAll()
		ln.tsLock.Unlock()
		for _, p := range peers {
			logging.LogInfo("Sync with ", p.Address)
			ln.tsLock.Lock()
			hashes := ln.searchDB.GetAllHashes()
			ln.tsLock.Unlock()
			p.Handshake(ln.proxySettings, ln.SelfPeer)
			newSearches := p.GetResultsForSync(ln.proxySettings, hashes)
			logging.LogTrace("Received", len(newSearches), "searches")
			newPeers := p.GetPeersForSync(ln.proxySettings)
			logging.LogTrace("Received", len(newPeers), "peers")

			ln.tsLock.Lock()
			ln.searchDB.SyncFrom(newSearches)
			ln.peerDB.SyncFrom(newPeers)
			ln.peerDB.UpdateRank(p)
			ln.tsLock.Unlock()
		}
	}
}
