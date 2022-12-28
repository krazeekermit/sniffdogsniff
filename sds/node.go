package main

import (
	"fmt"
	"sync"
	"time"
)

type LocalNode struct {
	tsLock        sync.Mutex // tread safe access from different threads, the NodeServer the WebUi, the SyncWithPeers()
	searchDB      SearchDB
	searchEngines map[string]SearchEngine
	minResultsThr int
	peerDB        PeerDB
	selfPeer      Peer
}

func InitNode(configs SdsConfig) LocalNode {
	var ln LocalNode
	ln.searchDB.Open(configs.searchDatabasePath)
	ln.peerDB.Open(configs.peersDatabasePath, configs.KnownPeers)
	ln.tsLock = sync.Mutex{}
	ln.searchEngines = configs.searchEngines
	ln.minResultsThr = 10 // 10 placeholder number will be defined in SdsConfigs
	ln.selfPeer = configs.NodePeerInfo
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

func (ln LocalNode) DoSearch(query string) []SearchResult {
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

func (ln LocalNode) SyncWithPeers() {
	for {
		time.Sleep(30 * time.Second)

		ln.tsLock.Lock()
		peers := ln.peerDB.GetAll()
		ln.tsLock.Unlock()
		for _, p := range peers {
			logInfo(fmt.Sprintf("Sync with %s", p.Address))
			ln.tsLock.Lock()
			hashes := ln.searchDB.GetAllHashes()
			ln.tsLock.Unlock()
			p.Handshake(ln.selfPeer)
			newSearches := p.GetResultsForSync(hashes)
			newPeers := p.GetPeersForSync()

			ln.tsLock.Lock()
			ln.searchDB.SyncFrom(newSearches)
			ln.peerDB.SyncFrom(newPeers)
			ln.peerDB.UpdateRank(p)
			ln.tsLock.Unlock()
		}
	}
}
