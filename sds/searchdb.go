package sds

// TABLE SEARCHES (RHASH, TIMESTAMP, URL, TITLE, DESCRIPTION)
// TABLE SEARCHES_META (RHASH, TIMESTAMP, SCORE, INVALIDATED)

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	//_ "github.com/go-gorp/gorp"

	_ "github.com/mattn/go-sqlite3"
	"github.com/sniffdogsniff/util"
	"github.com/sniffdogsniff/util/logging"
)

const SEARCHES_DB_FILE_NAME = "searches.db"

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

type searchDBPrivate struct {
	dbObject *sql.DB
}

func (sd *searchDBPrivate) Open(path string) {
	sql, err := sql.Open("sqlite3", path)
	if err != nil {
		logging.LogError(err.Error())
		return
	} else {
		sd.dbObject = sql
	}

	_, err = sql.Exec("create table SEARCHES(HASH text, TIMESTAMP bigint unsigned, TITLE text, URL text, DESCRIPTION text)")
	if err != nil {
		logging.LogTrace("DB size :: ", sd.GetDBSizeInBytes())
		logging.LogWarn(err.Error())
	}

	_, err = sql.Exec("create table SEARCHES_META(HASH text, TIMESTAMP bigint unsigned, SCORE int, INVALIDATED int)")
	if err != nil {
		logging.LogWarn(err.Error())
	}
}

func (sd *searchDBPrivate) HasHash(rHash string) bool {
	has, _ := sd.GetByHash(rHash)
	return has
}

func (sd *searchDBPrivate) GetByHash(rHash string) (bool, SearchResult) {
	query := sd.DoQuery(fmt.Sprintf("select * from SEARCHES where HASH == '%s'", rHash), -1)
	return len(query) > 0, query[util.B64UrlsafeStringToHash(rHash)]
}

func (sd *searchDBPrivate) GetMetaByHash(h [32]byte) (bool, ResultMeta) {
	query := sd.QueryMetaTable(fmt.Sprintf("select * from SEARCHES_META where HASH = '%s'",
		util.HashToB64UrlsafeString(h)), -1)
	return len(query) > 0, query[h]
}

func (sd *searchDBPrivate) GetLastTimestamp() uint64 {
	rows, err := sd.dbObject.Query("select max(TIMESTAMP) from SEARCHES")
	if err != nil {
		panic("SearchDB: Could not determine last timestamp: DB corruption")
	}
	var timestamp uint64
	for rows.Next() {
		rows.Scan(&timestamp)
	}
	return timestamp
}

/**
  * SEARCHES_META time stamps are mutable
  *
**/
func (sd *searchDBPrivate) GetLastMetaTimestamp() uint64 {
	rows, err := sd.dbObject.Query("select max(TIMESTAMP) from SEARCHES_META")
	if err != nil {
		panic("SearchDB: Could not determine last timestamp: DB corruption" + err.Error())
	}
	var timestamp uint64
	for rows.Next() {
		rows.Scan(&timestamp)
	}
	return timestamp
}

func (sd *searchDBPrivate) DoSearch(text string) map[[32]byte]SearchResult {
	/*
	 * ScoredSearchResult is a wrapper struct that helps sorting results by score
	 */
	type ScoredSearchResult struct {
		Result SearchResult
		Score  int
	}

	sqlString := fmt.Sprintf("select s1.HASH, TITLE, URL, DESCRIPTION, SCORE from (%s) as s1 join SEARCHES_META as s2 on s1.HASH = s2.HASH",
		buildSearchQuery(text))
	rows, err := sd.dbObject.Query(sqlString)
	if err != nil {
		return make(map[[32]byte]SearchResult, 0)
	}

	toSort := make([]ScoredSearchResult, 0)

	var b64Hash string
	var title string
	var url string
	var description string
	var score int

	for rows.Next() {
		err := rows.Scan(&b64Hash, &title, &url, &description, &score)

		if err != nil {
			logging.LogTrace(err.Error())
			continue
		}
		toSort = append(toSort, ScoredSearchResult{
			Result: SearchResult{
				ResultHash:  util.B64UrlsafeStringToHash(b64Hash),
				Url:         url,
				Title:       title,
				Description: description,
			}, Score: score,
		})

	}
	sort.Slice(toSort, func(i, j int) bool {
		return toSort[i].Score > toSort[j].Score
	})
	sorted := make(map[[32]byte]SearchResult, 0)
	for _, v := range toSort {
		sorted[v.Result.ResultHash] = v.Result
	}
	logging.LogInfo("SearchDB", len(sorted), "results found in decentralized database")
	return sorted
}

func (sd *searchDBPrivate) GetAll() map[[32]byte]SearchResult {
	return sd.DoQuery("select * from SEARCHES", -1)
}

func (sd *searchDBPrivate) GetAllHashes() [][32]byte {
	hashes := make([][32]byte, 0)
	for _, result := range sd.GetAll() {
		hashes = append(hashes, result.ResultHash)
	}
	return hashes
}

/*
hashes: hashes that the sync requesting peer has in its database
the result will be the difference between the results that the peer
already has and the results that the peer doesn't have
*/
func (sd *searchDBPrivate) GetForSync(timestamp uint64, sizeLimit int) map[[32]byte]SearchResult {
	return sd.DoQuery(fmt.Sprintf("select * from SEARCHES where TIMESTAMP > %d", timestamp), sizeLimit)
}

func (sd *searchDBPrivate) SyncFrom(results []SearchResult) {
	hashes := sd.GetAllHashes()
	for _, sr := range results {
		if !util.Array32Contains(hashes, sr.ResultHash) {
			if sr.IsConsistent() {
				sd.InsertRow(sr)
			}
		}
	}
}

func (sd *searchDBPrivate) GetAllMetadata() map[[32]byte]ResultMeta {
	return sd.QueryMetaTable("select * from SEARCHES_META", -1)
}

func (sd *searchDBPrivate) GetResultsMetadataForSync(ts uint64, sizeLimit int) map[[32]byte]ResultMeta {
	return sd.QueryMetaTable(fmt.Sprintf("select * from SEARCHES_META where TIMESTAMP > %d", ts), sizeLimit)
}

func (sd *searchDBPrivate) QueryMetaTable(query string, sizeLimit int) map[[32]byte]ResultMeta {
	rows, err := sd.dbObject.Query(query)

	metas := make(map[[32]byte]ResultMeta, 0)

	if err != nil {
		return metas
	}

	count := 0

	var b64Hash string
	var timestamp uint64
	var score int
	var invalidated Invalidation

	for rows.Next() {

		if sizeLimit > 0 && count >= sizeLimit {
			break
		}

		err := rows.Scan(&b64Hash, &timestamp, &score, &invalidated)

		if err != nil {
			logging.LogTrace(err.Error())
			continue
		}

		bHash := util.B64UrlsafeStringToHash(b64Hash)
		metas[bHash] = ResultMeta{
			ResultHash:  bHash,
			Timestamp:   timestamp,
			Score:       score,
			Invalidated: invalidated,
		}
		count++

	}
	return metas
}

func (sd *searchDBPrivate) SyncResultsMetadataFrom(metas []ResultMeta) {
	for _, mt := range metas {
		hash := util.HashToB64UrlsafeString(mt.ResultHash)
		// average score when sync - auto adjust: more data means more close to real value
		_, err := sd.dbObject.Exec(fmt.Sprintf("update SEARCHES_META set SCORE = (SCORE + %d / 2) where TIMESTAMP < %d and HASH = '%s'",
			mt.Score, mt.Timestamp, hash))
		if err != nil {
			logging.LogTrace(err.Error())
		}
		_, err = sd.dbObject.Exec(fmt.Sprintf("update SEARCHES_META set INVALIDATED = %d where HASH = '%s'",
			mt.Invalidated, hash))
		if err != nil {
			logging.LogTrace(err.Error())
		}
	}
}

func (sd *searchDBPrivate) InsertRow(sr SearchResult) {
	hashStr := util.HashToB64UrlsafeString(sr.ResultHash)
	_, err := sd.dbObject.Exec(fmt.Sprintf(
		"insert or ignore into SEARCHES values('%s', %d, '%s', '%s', '%s')",
		hashStr, sr.Timestamp, sr.Title, sr.Url, sr.Description))
	if err != nil {
		logging.LogTrace(err)
	}
	_, err = sd.dbObject.Exec(fmt.Sprintf(
		"insert or ignore into SEARCHES_META values('%s', %d, %d, %d)",
		hashStr, 0, 0, NONE))
	if err != nil {
		logging.LogTrace(err)
	}
}

func (sd *searchDBPrivate) DeleteInvalidated() {
	_, err := sd.dbObject.Exec(fmt.Sprintf(
		"delete from SEARCHES as s1 where s1.HASH in (select s2.HASH from SEARCHES_META as s2 where INVALIDATED = %d)", INVALIDATED))
	if err != nil {
		logging.LogTrace(err)
	}
	_, err = sd.dbObject.Exec(fmt.Sprintf(
		"delete from SEARCHES_META where INVALIDATED = %d", INVALIDATED))
	if err != nil {
		logging.LogTrace(err)
	}
}

func (sd *searchDBPrivate) UpdateResultScore(hash [32]byte, increment int) {
	_, err := sd.dbObject.Exec(fmt.Sprintf("update SEARCHES_META set TIMESTAMP = %d, SCORE = SCORE + %d where HASH = '%s'",
		time.Now().Unix(), increment, util.HashToB64UrlsafeString(hash)))
	if err != nil {
		logging.LogTrace(err.Error())
	}
}

func (sd *searchDBPrivate) SetInvalidationLevel(hash [32]byte, level Invalidation) {
	_, err := sd.dbObject.Exec(fmt.Sprintf("update SEARCHES_META set INVALIDATED = %d where HASH = '%s'",
		level, util.HashToB64UrlsafeString(hash)))
	if err != nil {
		logging.LogTrace(err.Error())
	}
}

func (sd *searchDBPrivate) DoQuery(queryString string, sizeLimit int) map[[32]byte]SearchResult {
	rows, err := sd.dbObject.Query(queryString)

	results := make(map[[32]byte]SearchResult, 0)

	if err != nil {
		logging.LogError("SearchDB", "Query:", queryString, err.Error())
		return results
	}

	count := 0

	var b64Hash string
	var timestamp uint64
	var title string
	var url string
	var description string

	for rows.Next() {

		if sizeLimit != -1 && count >= sizeLimit {
			break
		}

		err := rows.Scan(&b64Hash, &timestamp, &title, &url, &description)
		if err != nil {
			logging.LogTrace(err.Error())
			continue
		}

		bHash := util.B64UrlsafeStringToHash(b64Hash)
		results[bHash] = SearchResult{
			ResultHash:  bHash,
			Timestamp:   timestamp,
			Url:         url,
			Title:       title,
			Description: description,
		}
		count++
	}
	return results
}

func (sd *searchDBPrivate) ClearTables() {
	for _, tn := range []string{"SEARCHES", "SEARCHES_META"} {
		_, err := sd.dbObject.Exec(fmt.Sprintf("delete from %s", tn))
		if err != nil {
			logging.LogError(err.Error())
		}
	}
}

func (sd *searchDBPrivate) GetEntriesCount() int {
	res, err := sd.dbObject.Query("select count(*) from SEARCHES")
	if err != nil {
		return 0
	}
	var nEntries int
	for res.Next() {
		res.Scan(&nEntries)
	}
	return nEntries
}

func (sd *searchDBPrivate) GetDBSizeInBytes() int {
	res, err := sd.dbObject.Query("PRAGMA PAGE_SIZE;")
	if err != nil {
		return -1
	}
	var pageSize int
	for res.Next() {
		res.Scan(&pageSize)
	}
	res, err = sd.dbObject.Query("PRAGMA PAGE_COUNT;")
	if err != nil {
		return -1
	}
	var pageCount int
	for res.Next() {
		res.Scan(&pageCount)
	}

	return pageSize * pageCount
}

/**
 * SearchDB is a wrapper containing the db (actual db on disk) and cache (a cached db stored in ram)
 * Sync operations GetResults Sync Results ecct are expensive in terms of time because they
 * read and write to the disk, so we decided to add an in-memory database to store the data temporarly
 * and then flush to disk when the cache size reaches its limit
**/

type SearchDB struct {
	maximumCacheSize  int
	db                *searchDBPrivate
	cache             *searchDBPrivate
	LastTimestamp     uint64
	LastMetaTimestamp uint64
	flushed           bool
}

func (sdb *SearchDB) Open(workDir string, maxCacheSize int) {
	sdb.maximumCacheSize = maxCacheSize
	logging.LogTrace("DEBUG: SearchDB started with", maxCacheSize, "Bytes of cache")
	sdb.db = new(searchDBPrivate)
	sdb.cache = new(searchDBPrivate)
	sdb.flushed = false
	sdb.db.Open(filepath.Join(workDir, SEARCHES_DB_FILE_NAME))
	sdb.cache.Open("file::memory:?cache=shared") //in ram database cache
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
