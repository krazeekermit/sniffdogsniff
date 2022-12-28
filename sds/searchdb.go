package main

// TABLE (RHASH, URL, TITLE, DESCRIPTION)
/*
	change design to not include the score (see python implementation)
	in search DB to keep it smaller
*/

import (
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"fmt"

	//_ "github.com/go-gorp/gorp"
	_ "github.com/mattn/go-sqlite3"
)

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
		m3_bytes = append(m3_bytes, array32ToSlice(sha256.Sum256([]byte(s)))...)
	}
	return sha256.Sum256(m3_bytes)
}

func (sr SearchResult) IsConsistent() bool {
	return sr.ResultHash == sr.calculateHash()
}

func (sr *SearchResult) ReHash() {
	sr.ResultHash = sr.calculateHash()
}

type SearchDB struct {
	dbObject *sql.DB
}

func (sd *SearchDB) Open(path string) {
	sql, err := sql.Open("sqlite3", path)
	if err != nil {
		logError(err.Error())
		return
	} else {
		sd.dbObject = sql
	}
	_, err = sql.Exec("create table SEARCHES(HASH text, TITLE text, URL text, DESCRIPTION text)")
	if err != nil {
		logWarn(err.Error())
	}

}

func (sd SearchDB) HasHash(rHash string) bool {
	has, _ := sd.GetByHash(rHash)
	return has
}

func (sd SearchDB) GetByHash(rHash string) (bool, SearchResult) {
	query := sd.DoQuery(fmt.Sprintf("select * from SEARCHES where RHASH == '%s'", rHash))
	return len(query) > 0, query[0]
}

func (sd SearchDB) DoSearch(text string) []SearchResult {
	queryString := fmt.Sprintf("select * from SEARCHES where TITLE like '%s' or URL like '%s' or DESCRIPTION like '%s'", text, text, text)
	query := sd.DoQuery(queryString)
	logInfo(fmt.Sprintf("SearchDB query results=%d", len(query)))
	return query
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
		if Array32Contains(hashes, result.ResultHash) {
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
		if !Array32Contains(hashes, sr.ResultHash) {
			if sr.IsConsistent() {
				sd.InsertRow(sr)
			}
		}
	}
}

func (sd SearchDB) InsertRow(sr SearchResult) {
	sd.dbObject.Exec(fmt.Sprintf(
		"insert or ignore into SEARCHES values('%s', '%s', '%s', '%s')",
		hashToB64UrlsafeString(sr.ResultHash), sr.Title, sr.Url, sr.Description))
}

func (sd SearchDB) DoQuery(queryString string) []SearchResult {
	rows, err := sd.dbObject.Query(queryString)
	if err != nil {
		logError(fmt.Sprintf("SearchDB %s", err.Error()))
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
			continue
		}
		results = append(results, SearchResult{
			ResultHash:  b64UrlsafeStringToHash(b64Hash),
			Url:         url,
			Title:       title,
			Description: description,
		})
	}
	return results
}

func hashToB64UrlsafeString(hash [32]byte) string {
	return base64.URLEncoding.EncodeToString(hash[:])
}

func b64UrlsafeStringToHash(b64 string) [32]byte {
	bytes, _ := base64.URLEncoding.DecodeString(b64)
	return SliceToArray32(bytes)
}
