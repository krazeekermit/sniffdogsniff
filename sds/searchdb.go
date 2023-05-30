package sds

// TABLE SEARCHES (RHASH, TIMESTAMP, URL, TITLE, DESCRIPTION)
// TABLE SEARCHES_META (RHASH, TIMESTAMP, SCORE, INVALIDATED)

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	//_ "github.com/go-gorp/gorp"

	"github.com/sniffdogsniff/util"
	"github.com/sniffdogsniff/util/logging"
	"github.com/syndtr/goleveldb/leveldb"
)

const SEARCHES_DB_FILE_NAME = "searches.db"
const METASS_DB_FILE_NAME = "searches_meta.db"

const ENTRY_MAX_SIZE = 512 // bytes

func buildSearchQuery(text string) string {
	queryString := fmt.Sprint("select * from SEARCHES where TITLE like '%", text, "%' or URL like '%", text, "%' or DESCRIPTION like '%", text, "%'")
	tokens := strings.Split(text, " ")
	if len(tokens) == 1 {
		return queryString
	}
	for _, token := range tokens {
		queryString += " union "
		queryString += fmt.Sprint("select * from SEARCHES where TITLE like '%", token, "%' or URL like '%", token, "%' or DESCRIPTION like '%", token, "%'")
	}
	logging.LogTrace(queryString)
	return queryString
}

type SearchResult struct {
	ResultHash  [32]byte
	Timestamp   uint64
	Title       string
	Url         string
	Description string
}

/*
SearchResult Structure
[[HASH (32)][TIMESTAMP (64)][TITLE][URL][DESCRIPTION]] = 512 bytes
*/
func NewSearchResult(title, url, description string) SearchResult {
	if 32+64+len(title)+len(url)+len(description) > ENTRY_MAX_SIZE {
		description = string(description[:(ENTRY_MAX_SIZE - 32 - 64 - len(title) - len(url) - 1)])
	}
	rs := SearchResult{Timestamp: uint64(time.Now().Unix()), Title: title, Url: url, Description: description}
	rs.ReHash()
	return rs
}

func searchResultFromBytes(bytez []byte) SearchResult {
	buf := bytes.NewBuffer(bytez)
	buf.Read
	hash := util.SliceToArray32(bytez[0:32])
	titleLen := uint8(bytez[33])
	title := string(bytez[40 : titleLen-1])
	urlLen := uint8(bytez[titleLen-1])
}

func (sr SearchResult) calculateHash() [32]byte {
	m3_bytes := make([]byte, 0)

	for _, s := range []string{sr.Url, sr.Title, sr.Description} {
		m3_bytes = append(m3_bytes, util.Array32ToSlice(sha256.Sum256([]byte(s)))...)
	}
	return sha256.Sum256(m3_bytes)
}

func (sr SearchResult) IsConsistent() bool {
	return sr.ResultHash == sr.calculateHash()
}

func (sr *SearchResult) ReHash() {
	sr.ResultHash = sr.calculateHash()
}

func (sr *SearchResult) HashAsB64UrlSafeStr() string {
	return util.HashToB64UrlsafeString(sr.ResultHash)
}

type Invalidation int

const (
	NONE        Invalidation = 0
	PENDING     Invalidation = 1
	INVALIDATED Invalidation = 2
)

type ResultMeta struct {
	ResultHash  [32]byte
	Timestamp   uint64
	Score       int
	Invalidated Invalidation
}

func NewResultMetadata(hash [32]byte, score int, invalidated Invalidation) ResultMeta {
	return ResultMeta{ResultHash: hash, Timestamp: uint64(time.Now().Unix()), Score: score, Invalidated: invalidated}
}

/****************************************************************************************************/

func openDB(dbPath string) *leveldb.DB {
	db, err := leveldb.OpenFile(dbPath, nil)
	if err != nil {
		panic(err.Error())
	}
	return db
}

/**
 * SearchDB is a wrapper containing the db (actual db on disk) and cache (a cached db stored in ram)
 * Sync operations GetResults Sync Results ecct are expensive in terms of time because they
 * read and write to the disk, so we decided to add an in-memory database to store the data temporarly
 * and then flush to disk when the cache size reaches its limit
**/

type SearchDB struct {
	maximumCacheSize  int
	searchesDB        *leveldb.DB
	metasDB           *leveldb.DB
	searchesCache     map[[32]byte]SearchResult
	metasCache        map[[32]byte]ResultMeta
	LastTimestamp     uint64
	LastMetaTimestamp uint64
	flushed           bool
}

func (sdb *SearchDB) Open(workDir string, maxCacheSize int) {
	sdb.maximumCacheSize = maxCacheSize
	logging.LogTrace("DEBUG: SearchDB started with", maxCacheSize, "Bytes of cache")

	sdb.searchesDB = openDB(filepath.Join(workDir, SEARCHES_DB_FILE_NAME))
	sdb.metasDB = openDB(filepath.Join(workDir, SEARCHES_DB_FILE_NAME))

	// last timestamp is initially set to the last timestamp of the on-disk db
	sdb.LastTimestamp = sdb.db.GetLastTimestamp()
	sdb.LastMetaTimestamp = sdb.db.GetLastMetaTimestamp()
}

func (sdb *SearchDB) IsEmpty() bool {
	return sdb.db.GetEntriesCount() == 0 && sdb.cache.GetEntriesCount() == 0
}

func (sdb *SearchDB) GetMemCacheApproxSize() int {
	return sdb.cache.GetDBSizeInBytes()
}

func (sdb *SearchDB) DoSearch(text string) []SearchResult {
	return util.MapToSlice(
		util.MergeMaps(sdb.db.DoSearch(text), sdb.cache.DoSearch(text)))
}

func (sdb *SearchDB) GetAllHashes() [][32]byte {
	hashes := sdb.db.GetAllHashes()
	hashes = append(hashes, sdb.cache.GetAllHashes()...)
	return hashes
}

func (sdb *SearchDB) GetForSync(timestamp uint64, sizeLimit int) []SearchResult {
	dbMap := sdb.db.GetForSync(timestamp, sizeLimit)
	cacheMap := sdb.cache.GetForSync(timestamp, sizeLimit-len(dbMap))
	return util.MapToSlice(util.MergeMaps(dbMap, cacheMap))
}

func (sdb *SearchDB) SyncFrom(results []SearchResult) {
	sdb.cache.SyncFrom(results)
	sdb.setUpdated()
}

func (sdb *SearchDB) GetMetadataOf(hashes [][32]byte) []ResultMeta {
	metas := make(map[[32]byte]ResultMeta, 0)
	for _, h := range hashes {
		present, meta := sdb.cache.GetMetaByHash(h)
		if present {
			metas[meta.ResultHash] = meta
		}
	}
	for _, h := range hashes {
		present, meta := sdb.db.GetMetaByHash(h)
		if present {
			metas[meta.ResultHash] = meta
		}
	}
	return util.MapToSlice(metas)
}

func (sdb *SearchDB) GetMetadataForSync(ts uint64, sizeLimit int) []ResultMeta {
	dbMap := sdb.db.GetResultsMetadataForSync(ts, sizeLimit)
	cacheMap := sdb.cache.GetResultsMetadataForSync(ts, sizeLimit-len(dbMap))
	return util.MapToSlice(util.MergeMaps(dbMap, cacheMap))
}

func (sdb *SearchDB) SyncResultsMetadataFrom(metas []ResultMeta) {
	sdb.cache.SyncResultsMetadataFrom(metas)
	sdb.setUpdated()
}

func (sdb *SearchDB) InsertResult(sr SearchResult) {
	sdb.cache.InsertRow(sr)
	sdb.setUpdated()
}

func (sdb *SearchDB) UpdateResultScore(hash [32]byte, increment int) {
	sdb.cache.UpdateResultScore(hash, increment)
	sdb.setUpdated()
}

func (sdb *SearchDB) setUpdated() {
	sdb.flushed = false
	searchesLastTime := sdb.cache.GetLastTimestamp()
	if searchesLastTime > 0 && searchesLastTime > sdb.LastTimestamp {
		sdb.LastTimestamp = searchesLastTime
	}
	metasLastTime := sdb.cache.GetLastMetaTimestamp()
	if metasLastTime > 0 && metasLastTime > sdb.LastMetaTimestamp {
		sdb.LastMetaTimestamp = metasLastTime
	}
	if sdb.cache.GetDBSizeInBytes() >= int(sdb.maximumCacheSize) {
		sdb.Flush()
	}
}

func (sdb *SearchDB) GetCacheDB() *searchDBPrivate {
	return sdb.cache
}

func (sdb *SearchDB) GetDB() *searchDBPrivate {
	return sdb.db
}

func (sdb *SearchDB) InvalidateResult(rHash [32]byte) {
	sdb.cache.SetInvalidationLevel(rHash, INVALIDATED)
}

func (sdb *SearchDB) Flush() {
	if sdb.flushed {
		return
	}

	/*
	 * when syncing from ram database to normal file database it immediatly flushes to disk
	 */
	sdb.db.SyncFrom(util.MapToSlice(sdb.cache.GetAll()))
	sdb.db.SyncResultsMetadataFrom(util.MapToSlice(sdb.cache.GetAllMetadata()))

	sdb.db.DeleteInvalidated()

	sdb.LastTimestamp = sdb.db.GetLastTimestamp()
	sdb.LastMetaTimestamp = sdb.db.GetLastMetaTimestamp()

	sdb.cache.ClearTables()
	sdb.flushed = true
}
