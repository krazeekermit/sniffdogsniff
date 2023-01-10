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

type SearchDB struct {
	dbObject *sql.DB
}

func (sd *SearchDB) Open(path string) {
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

func (sd SearchDB) HasHash(rHash string) bool {
	has, _ := sd.GetByHash(rHash)
	return has
}

func (sd SearchDB) GetByHash(rHash string) (bool, SearchResult) {
	query := sd.DoQuery(fmt.Sprintf("select * from SEARCHES where HASH == '%s'", rHash))
	return len(query) > 0, query[0]
}

func (sd SearchDB) DoSearch(text string) []SearchResult {
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
		return make([]SearchResult, 0)
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
	sorted := make([]SearchResult, 0)
	for _, v := range toSort {
		sorted = append(sorted, v.Result)
	}
	logging.LogInfo("SearchDB", len(sorted), "results found in decentralized database")
	return sorted
}

func (sd SearchDB) GetAll() []SearchResult {
	return sd.DoQuery("select * from SEARCHES")
}

func (sd SearchDB) GetAllHashes() [][32]byte {
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
func (sd SearchDB) GetForSync(hashes [][32]byte) []SearchResult {
	results := make([]SearchResult, 0)
	for _, result := range sd.GetAll() {
		if util.Array32Contains(hashes, result.ResultHash) {
			continue
		} else {
			results = append(results, result)
		}
	}
	return results
}

func (sd SearchDB) SyncFrom(results []SearchResult) {
	hashes := sd.GetAllHashes()
	for _, sr := range results {
		if !util.Array32Contains(hashes, sr.ResultHash) {
			if sr.IsConsistent() {
				sd.InsertRow(sr)
			}
		}
	}
}

func (sd SearchDB) GetAllResultsMetadata() []ResultMeta {
	rows, err := sd.dbObject.Query("select * from SEARCHES_META")
	if err != nil {
		return make([]ResultMeta, 0)
	}

	metas := make([]ResultMeta, 0)

	var b64Hash string
	var score int
	var invalidated int

	for rows.Next() {
		err := rows.Scan(&b64Hash, &score, &invalidated)

		if err != nil {
			logging.LogTrace(err.Error())
			continue
		}
		metas = append(metas, ResultMeta{
			ResultHash:  util.B64UrlsafeStringToHash(b64Hash),
			Score:       score,
			Invalidated: invalidated,
		})

	}
	return metas
}

func (sd SearchDB) SyncResultsMetadataFrom(metas []ResultMeta) {
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

func (sd SearchDB) InsertRow(sr SearchResult) {
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

func (sd SearchDB) UpdateResultScore(hash [32]byte, increment int) {
	_, err := sd.dbObject.Exec(fmt.Sprintf("update SEARCHES_META set SCORE = SCORE + %d where HASH = '%s'",
		increment, util.HashToB64UrlsafeString(hash)))
	if err != nil {
		logging.LogTrace(err.Error())
	}
}

func (sd SearchDB) DoQuery(queryString string) []SearchResult {
	rows, err := sd.dbObject.Query(queryString)
	if err != nil {
		logging.LogError("SearchDB", "Query:", queryString, err.Error())
		return make([]SearchResult, 0)
	}

	results := make([]SearchResult, 0)

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
		results = append(results, SearchResult{
			ResultHash:  util.B64UrlsafeStringToHash(b64Hash),
			Url:         url,
			Title:       title,
			Description: description,
		})
	}
	return results
}
