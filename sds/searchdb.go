package sds

// TABLE SEARCHES (RHASH, URL, TITLE, DESCRIPTION)
// TABLE SEARCHES_META (RHASH, SCORE, INVALIDATED)
/*
	change design to not include the score (see python implementation)
	in search DB to keep it smaller
*/

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	//_ "github.com/go-gorp/gorp"
	_ "github.com/mattn/go-sqlite3"
	"gitlab.com/sniffdogsniff/util"
	"gitlab.com/sniffdogsniff/util/logging"
)

const MAX_RAM_DB_SIZE = 268435456 // 256 MB

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
	Title       string
	Url         string
	Description string
}

func NewSearchResult(title, url, description string) SearchResult {
	rs := SearchResult{Title: title, Url: url, Description: description}
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

type ResultMeta struct {
	ResultHash  [32]byte
	Score       int
	Invalidated int
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
	_, err = sql.Exec("create table SEARCHES(HASH text, TITLE text, URL text, DESCRIPTION text)")
	if err != nil {
		logging.LogWarn(err.Error())
	}
	_, err = sql.Exec("create table SEARCHES_META(HASH text, SCORE int, INVALIDATED int)")

	if err != nil {
		logging.LogWarn(err.Error())
	}
}

func (sd *searchDBPrivate) HasHash(rHash string) bool {
	has, _ := sd.GetByHash(rHash)
	return has
}

func (sd *searchDBPrivate) GetByHash(rHash string) (bool, SearchResult) {
	query := sd.DoQuery(fmt.Sprintf("select * from SEARCHES where HASH == '%s'", rHash))
	return len(query) > 0, query[util.B64UrlsafeStringToHash(rHash)]
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
	return sd.DoQuery("select * from SEARCHES")
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
func (sd *searchDBPrivate) GetForSync(hashes [][32]byte) map[[32]byte]SearchResult {
	results := make(map[[32]byte]SearchResult, 0)
	for k, result := range sd.GetAll() {
		if util.Array32Contains(hashes, k) {
			continue
		} else {
			results[k] = result
		}
	}
	return results
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

func (sd *searchDBPrivate) GetAllResultsMetadata() map[[32]byte]ResultMeta {
	rows, err := sd.dbObject.Query("select * from SEARCHES_META")

	metas := make(map[[32]byte]ResultMeta, 0)

	if err != nil {
		return metas
	}

	var b64Hash string
	var score int
	var invalidated int

	for rows.Next() {
		err := rows.Scan(&b64Hash, &score, &invalidated)

		if err != nil {
			logging.LogTrace(err.Error())
			continue
		}

		bHash := util.B64UrlsafeStringToHash(b64Hash)
		metas[bHash] = ResultMeta{
			ResultHash:  bHash,
			Score:       score,
			Invalidated: invalidated,
		}

	}
	return metas
}

func (sd *searchDBPrivate) SyncResultsMetadataFrom(metas []ResultMeta) {
	for _, mt := range metas {
		hash := util.HashToB64UrlsafeString(mt.ResultHash)
		// average score when sync - auto adjust: more data means more close to real value
		_, err := sd.dbObject.Exec(fmt.Sprintf("update SEARCHES_META set SCORE = SCORE + %d / 2 where HASH = '%s'",
			mt.Score, hash))
		if err != nil {
			logging.LogTrace(err.Error())
		}
		// link invalidation not supported for now, invalidation requires to solve a distributed sort-of consensus
		// problem and this is not simple (for now trying to take inspiration from poker game)
		// _, err = sd.dbObject.Exec(fmt.Sprintf("update SEARCHES_META set INVALIDATED = INVALIDATED + %d / 2 where HASH = '%s'",
		// 	mt.Invalidated, hash))
		// if err != nil {
		// 	logging.LogTrace(err.Error())
		// }
	}
}

func (sd *searchDBPrivate) InsertRow(sr SearchResult) {
	hashStr := util.HashToB64UrlsafeString(sr.ResultHash)
	_, err := sd.dbObject.Exec(fmt.Sprintf(
		"insert or ignore into SEARCHES values('%s', '%s', '%s', '%s')",
		hashStr, sr.Title, sr.Url, sr.Description))
	if err != nil {
		logging.LogTrace(err)
	}
	_, err = sd.dbObject.Exec(fmt.Sprintf(
		"insert or ignore into SEARCHES_META values('%s', %d, %d)",
		hashStr, 0, 0))
	if err != nil {
		logging.LogTrace(err)
	}
}

func (sd *searchDBPrivate) UpdateResultScore(hash [32]byte, increment int) {
	_, err := sd.dbObject.Exec(fmt.Sprintf("update SEARCHES_META set SCORE = SCORE + %d where HASH = '%s'",
		increment, util.HashToB64UrlsafeString(hash)))
	if err != nil {
		logging.LogTrace(err.Error())
	}
}

func (sd *searchDBPrivate) DoQuery(queryString string) map[[32]byte]SearchResult {
	rows, err := sd.dbObject.Query(queryString)

	results := make(map[[32]byte]SearchResult, 0)

	if err != nil {
		logging.LogError("SearchDB", "Query:", queryString, err.Error())
		return results
	}

	var b64Hash string
	var title string
	var url string
	var description string

	for rows.Next() {
		err := rows.Scan(&b64Hash, &title, &url, &description)

		if err != nil {
			logging.LogTrace(err.Error())
			continue
		}

		bHash := util.B64UrlsafeStringToHash(b64Hash)
		results[bHash] = SearchResult{
			ResultHash:  bHash,
			Url:         url,
			Title:       title,
			Description: description,
		}
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

func (sd *searchDBPrivate) GetDBSizeInBytes() int {
	dbSize := 0
	res, err := sd.dbObject.Query("select sum(length(HASH)+length(TITLE)+length(URL)+length(DESCRIPTION)) from SEARCHES;")
	if err != nil {
		logging.LogTrace(err.Error())
		return -1
	}
	var ts int
	for res.Next() {
		res.Scan(&ts)
	}
	dbSize += ts
	res, err = sd.dbObject.Query("select sum(length(HASH)+length(SCORE)+length(INVALIDATED)) from SEARCHES_META;")
	if err != nil {
		logging.LogTrace(err.Error())
		return -1
	}
	for res.Next() {
		res.Scan(&ts)
	}
	dbSize += ts

	return dbSize
}

/**
 * SearchDB is a wrapper containing the db (actual db on disk) and cache (a cached db stored in ram)
 * Sync operations GetHashes, GetResults Sync Results ecct are expensive in terms of time because they
 * read and write to the disk, so we decided to add an in-memory database to store the data temporarly
 * and then flush to disk
**/

type SearchDB struct {
	db      *searchDBPrivate
	cache   *searchDBPrivate
	flushed bool
}

func (sdb *SearchDB) Open(path string) {
	sdb.db = new(searchDBPrivate)
	sdb.cache = new(searchDBPrivate)
	sdb.flushed = false
	sdb.db.Open(path)
	sdb.cache.Open("file::memory:?cache=shared") //in ram database cache
}

func (sdb *SearchDB) GetMemCacheApproxSize() int {
	return sdb.cache.GetDBSizeInBytes()
}

func (sdb *SearchDB) DoSearch(text string) []SearchResult {
	return util.MapToSlice[[32]byte, SearchResult](
		util.MergeMaps[[32]byte, SearchResult](sdb.db.DoSearch(text), sdb.cache.DoSearch(text)))
}

func (sdb *SearchDB) GetAllHashes() [][32]byte {
	hashes := sdb.db.GetAllHashes()
	hashes = append(hashes, sdb.cache.GetAllHashes()...)
	return hashes
}

func (sdb *SearchDB) GetForSync(hashes [][32]byte) []SearchResult {
	return util.MapToSlice[[32]byte, SearchResult](
		util.MergeMaps[[32]byte, SearchResult](sdb.db.GetForSync(hashes), sdb.cache.GetForSync(hashes)))
}

func (sdb *SearchDB) SyncFrom(results []SearchResult) {
	sdb.cache.SyncFrom(results)
	fmt.Println(len(sdb.cache.GetAll()))
	sdb.setUpdated()
	fmt.Println(len(sdb.cache.GetAll()))
}

func (sdb *SearchDB) GetAllResultsMetadata() []ResultMeta {
	return util.MapToSlice[[32]byte, ResultMeta](
		util.MergeMaps[[32]byte, ResultMeta](sdb.db.GetAllResultsMetadata(), sdb.cache.GetAllResultsMetadata()))
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
	if sdb.cache.GetDBSizeInBytes() >= MAX_RAM_DB_SIZE {
		sdb.FlushToDisk()
	}
}

func (sdb *SearchDB) FlushToDisk() {
	if sdb.flushed {
		return
	}

	/*
	 * when syncing from ram database to normal file database it immediatly flushes to disk
	 */
	sdb.db.SyncFrom(util.MapToSlice[[32]byte, SearchResult](sdb.cache.GetAll()))
	sdb.db.SyncResultsMetadataFrom(util.MapToSlice[[32]byte, ResultMeta](sdb.cache.GetAllResultsMetadata()))

	sdb.cache.ClearTables()
	sdb.flushed = true
}
