package sds

// TABLE SEARCHES (RHASH, TIMESTAMP, URL, TITLE, DESCRIPTION)
// TABLE SEARCHES_META (RHASH, TIMESTAMP, SCORE, INVALIDATED)

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/sniffdogsniff/util"
	"github.com/sniffdogsniff/util/logging"
	"github.com/syndtr/goleveldb/leveldb"
)

const SEARCHES_DB_FILE_NAME = "searches.db"
const METASS_DB_FILE_NAME = "searches_meta.db"

const SEARCH_RESULT_BYTE_SIZE = 512 // bytes
const RESULT_META_BYTE_SIZE = 43    // bytes

type ResultDataType uint8

const (
	LINK_DATA_TYPE  ResultDataType = 0
	IMAGE_DATA_TYPE ResultDataType = 1
	VIDEO_DATA_TYPE ResultDataType = 2
)

type ResultPropertiesMap map[uint8]string

const (
	RP_THUMB_LINK  uint8 = 0
	RP_SOURCE_LINK uint8 = 1
	RP_DESCRIPTION uint8 = 2
)

type SearchResult struct {
	ResultHash [32]byte
	Timestamp  uint64
	Title      string
	Url        string
	Properties ResultPropertiesMap
	DataType   ResultDataType
}

/*
SearchResult Structure
[[HASH (32)][TIMESTAMP (8)][TITLE][URL][PROPERTIES]] = 512 bytes
*/
func NewSearchResult(title, url string, properties ResultPropertiesMap, dataType ResultDataType) SearchResult {
	const FIXED_LENGHT = 32 + 8 + 1 + 1 + 1 + 1 // Hash=32, time=8, len(title)=1, len(url)=1, len(desc)=1, datatype=1
	// SHRINK TITLE AND PROPERTIES TO FIT 512 byte max size

	propertiesLen := 0
	for _, p := range properties {
		propertiesLen += 1 + len(p)
	}

	if FIXED_LENGHT+len(title)+len(url)+propertiesLen > SEARCH_RESULT_BYTE_SIZE {
		newPropertiesLen := SEARCH_RESULT_BYTE_SIZE - FIXED_LENGHT - len(title) - len(url)

		if newPropertiesLen < 0 {
			properties = make(ResultPropertiesMap, 0)
		} else {
			filledLen := 0
			for k, _ := range properties {
				plen := filledLen + len(properties[k]) + 2 //property lenght + 1 (key lenght) + strlen
				if plen <= newPropertiesLen {
					properties[k] = properties[k][:newPropertiesLen-2]
					filledLen += len(properties[k]) + 2
				}
			}
		}
		propertiesLen = newPropertiesLen
	}
	if FIXED_LENGHT+len(title)+len(url)+propertiesLen > SEARCH_RESULT_BYTE_SIZE {
		newTitleLen := SEARCH_RESULT_BYTE_SIZE - FIXED_LENGHT - len(url) - propertiesLen
		if newTitleLen < 0 {
			title = ""
		} else if newTitleLen <= len(title) {
			title = title[:newTitleLen]
		}
	}

	rs := SearchResult{
		Timestamp:  uint64(time.Now().Unix()),
		Title:      title,
		Url:        url,
		DataType:   dataType,
		Properties: properties,
	}
	rs.ReHash()
	return rs
}

func BytesToSearchResult(hash [32]byte, bytez []byte) (SearchResult, error) {
	buf := bytes.NewBuffer(bytez)

	bts := make([]byte, 8) // TIMESTAMP
	n, err := buf.Read(bts)
	if err != nil || n != 8 {
		return SearchResult{}, err
	}

	titleLen, err := buf.ReadByte() // TITLE
	if err != nil {
		return SearchResult{}, err
	}
	btitle := make([]byte, titleLen)
	n, err = buf.Read(btitle)
	if err != nil || n != int(titleLen) {
		return SearchResult{}, err
	}

	urlLen, err := buf.ReadByte() // URL
	if err != nil {
		return SearchResult{}, err
	}
	burl := make([]byte, urlLen)
	n, err = buf.Read(burl)
	if err != nil || n != int(urlLen) {
		return SearchResult{}, err
	}

	propsLen, err := buf.ReadByte() // PROPERTIES
	if err != nil {
		return SearchResult{}, err
	}

	properties := make(ResultPropertiesMap)
	for i := 0; i < int(propsLen); i++ {
		propKey, err := buf.ReadByte()
		if err != nil {
			continue
		}
		propLen, err := buf.ReadByte()
		if err != nil {
			continue
		}
		bprop := make([]byte, propLen)
		n, err = buf.Read(bprop)
		if err != nil || n != int(propLen) {
			continue
		}
		properties[uint8(propKey)] = string(bprop)
	}

	dataType, err := buf.ReadByte() // DATA_TYPe
	if err != nil {
		return SearchResult{}, err
	}

	return SearchResult{
		ResultHash: hash,
		Timestamp:  binary.LittleEndian.Uint64(bts),
		Title:      string(btitle),
		Url:        string(burl),
		Properties: properties,
		DataType:   ResultDataType(dataType),
	}, nil
}

func (sr *SearchResult) ToBytes() []byte {
	buf := bytes.NewBuffer(nil)
	bts := make([]byte, 8)
	binary.LittleEndian.PutUint64(bts, sr.Timestamp)
	buf.Write(bts)
	buf.WriteByte(byte(len(sr.Title)))
	buf.Write([]byte(sr.Title))
	buf.WriteByte(byte(len(sr.Url)))
	buf.Write([]byte(sr.Url))
	buf.WriteByte(byte(len(sr.Properties)))
	for k, p := range sr.Properties {
		buf.WriteByte(byte(k))
		buf.WriteByte(byte(len(p)))
		buf.Write([]byte(p))
	}
	buf.WriteByte(byte(sr.DataType))
	return buf.Bytes()
}

func (sr *SearchResult) calculateHash() [32]byte {
	m3_bytes := make([]byte, 0)
	for _, s := range []string{sr.Url, sr.Title} {
		m3_bytes = append(m3_bytes, util.Array32ToSlice(sha256.Sum256([]byte(s)))...)
	}
	for k, p := range sr.Properties {
		pb := make([]byte, 0)
		pb = append(pb, k)
		pb = append(pb, []byte(p)...)
		m3_bytes = append(m3_bytes, util.Array32ToSlice(sha256.Sum256([]byte(pb)))...)
	}

	dtb := make([]byte, 1)
	dtb[0] = byte(sr.DataType)
	m3_bytes = append(m3_bytes, util.Array32ToSlice(sha256.Sum256([]byte(dtb)))...)

	return sha256.Sum256(m3_bytes)
}

func (sr *SearchResult) IsConsistent() bool {
	return sr.ResultHash == sr.calculateHash()
}

func (sr *SearchResult) ReHash() {
	sr.ResultHash = sr.calculateHash()
}

func (sr *SearchResult) HashAsB64UrlSafeStr() string {
	return util.HashToB64UrlsafeString(sr.ResultHash)
}

func (sr *SearchResult) SafeGetProperty(propKey uint8) string {
	property, ok := sr.Properties[propKey]
	if ok {
		return property
	}
	return ""
}

type Invalidation uint8

const (
	NONE        Invalidation = 0
	PENDING     Invalidation = 1
	INVALIDATED Invalidation = 2
)

type ResultMeta struct {
	ResultHash  [32]byte
	Timestamp   uint64
	Score       uint16
	Invalidated Invalidation // int8
}

func NewResultMeta(hash [32]byte, ts uint64, score uint16, inv Invalidation) ResultMeta {
	return ResultMeta{
		ResultHash:  hash,
		Timestamp:   ts,
		Score:       score,
		Invalidated: inv,
	}
}

func BytesToResultMeta(hash [32]byte, bytez []byte) (ResultMeta, error) {
	buf := bytes.NewBuffer(bytez)

	bts := make([]byte, 8) // TIMESTAMP
	n, err := buf.Read(bts)
	if err != nil || n != 8 {
		return ResultMeta{}, err
	}

	bscore := make([]byte, 2) // SCORE
	n, err = buf.Read(bscore)
	if err != nil || n != 2 {
		return ResultMeta{}, err
	}

	inv, err := buf.ReadByte() // INVALIDATION
	if err != nil {
		return ResultMeta{}, err
	}

	return ResultMeta{
		ResultHash:  hash,
		Timestamp:   binary.LittleEndian.Uint64(bts),
		Score:       binary.LittleEndian.Uint16(bscore),
		Invalidated: Invalidation(inv),
	}, nil
}

func (rm *ResultMeta) UpdateTime() {
	rm.Timestamp = uint64(time.Now().Unix())
}

func (rm *ResultMeta) ToBytes() []byte {
	buf := bytes.NewBuffer(nil)
	bts := make([]byte, 8)
	binary.LittleEndian.PutUint64(bts, rm.Timestamp)
	buf.Write(bts)
	scs := make([]byte, 2)
	binary.LittleEndian.PutUint16(scs, rm.Score)
	buf.Write(scs)
	buf.WriteByte(byte(rm.Invalidated))
	return buf.Bytes()
}

/****************************************************************************************************/

/**
* SearchDB is containing the db (actual db on disk) and cache (a cached db stored in ram)
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

func openDB(dbPath string) *leveldb.DB {
	db, err := leveldb.OpenFile(dbPath, nil)
	if err != nil {
		panic(err.Error())
	}
	return db
}

func dbSize(db *leveldb.DB) int {
	it := db.NewIterator(nil, nil)
	size := 0
	for it.Next() {
		size++
	}
	it.Release()
	return size
}

func (sdb *SearchDB) Open(workDir string, maxCacheSize int) {
	sdb.maximumCacheSize = maxCacheSize
	logging.LogTrace("DEBUG: SearchDB started with", maxCacheSize, "Bytes of cache")

	sdb.searchesDB = openDB(filepath.Join(workDir, SEARCHES_DB_FILE_NAME))
	sdb.metasDB = openDB(filepath.Join(workDir, METASS_DB_FILE_NAME))

	// Initialize cache
	sdb.searchesCache = make(map[[32]byte]SearchResult, 0)
	sdb.metasCache = make(map[[32]byte]ResultMeta, 0)

	// last timestamp is initially set to the last timestamp of the on-disk db
	sdb.LastTimestamp = sdb.GetLastDBSearchesTimestamp()
	sdb.LastMetaTimestamp = sdb.GetLastDBMetasTimestamp()
}

func (sdb *SearchDB) GetLastDBSearchesTimestamp() uint64 {
	var lastts uint64
	lastts = 0
	iter := sdb.searchesDB.NewIterator(nil, nil)
	for iter.Next() {
		sr, err := BytesToSearchResult(util.SliceToArray32(iter.Key()), iter.Value())
		if err != nil {
			continue
		}
		if sr.Timestamp > lastts {
			lastts = sr.Timestamp
		}
	}
	iter.Release()
	return lastts
}

func (sdb *SearchDB) GetLastDBMetasTimestamp() uint64 {
	var lastts uint64
	lastts = 0
	iter := sdb.searchesDB.NewIterator(nil, nil)
	for iter.Next() {
		rm, err := BytesToResultMeta(util.SliceToArray32(iter.Key()), iter.Value())
		if err != nil {
			continue
		}
		if rm.Timestamp > lastts {
			lastts = rm.Timestamp
		}
	}
	iter.Release()
	return lastts
}

func (sdb *SearchDB) GetLastCachedSearchResultTimestamp() uint64 {
	var last uint64
	last = 0
	for _, sr := range sdb.searchesCache {
		if sr.Timestamp > last {
			last = sr.Timestamp
		}
	}
	return last
}

func (sdb *SearchDB) GetLastCachedResultMetaTimestamp() uint64 {
	var last uint64
	last = 0
	for _, rm := range sdb.metasCache {
		if rm.Timestamp > last {
			last = rm.Timestamp
		}
	}
	return last
}

func (sdb *SearchDB) IsEmpty() bool {
	return len(sdb.searchesCache) == 0 && dbSize(sdb.searchesDB) == 0
}

func matchesSearch(text string, sr SearchResult) bool {
	text = strings.ToLower(text)
	title := strings.ToLower(sr.Title)
	desc := strings.ToLower(sr.SafeGetProperty(RP_DESCRIPTION))
	if strings.Contains(title, text) || strings.Contains(desc, text) {
		return true
	}
	if strings.Contains(text, ".") && !strings.Contains(text, " ") && strings.Contains(sr.Url, text) {
		return true
	}
	toks := strings.Split(text, " ")
	count := 0
	for _, s := range toks {
		snorm := strings.TrimSpace(s)
		if strings.Contains(title, snorm) || strings.Contains(desc, snorm) {
			count++
		}
	}
	return count > int(len(toks)*75/100)
}

func (sdb *SearchDB) DoSearch(text string) []SearchResult {
	results := make([]SearchResult, 0)

	type scoredResult struct {
		score uint16
		sr    SearchResult
	}

	for _, sr := range sdb.searchesCache {
		if matchesSearch(text, sr) {
			results = append(results, sr)
		}
	}

	iter := sdb.searchesDB.NewIterator(nil, nil)
	for iter.Next() {
		sr, err := BytesToSearchResult(util.SliceToArray32(iter.Key()), iter.Value())
		if err != nil {
			continue
		}
		if matchesSearch(text, sr) {
			results = append(results, sr)
		}
	}
	iter.Release()

	// Retrieve Metadata score for sorting
	toSort := make([]scoredResult, 0)
	for _, sr := range results {
		var score uint16
		score = 0
		rm, present := sdb.metasCache[sr.ResultHash]
		if present {
			score = rm.Score
		} else {
			value, err := sdb.metasDB.Get(util.Array32ToSlice(sr.ResultHash), nil)
			if err != nil {
				rm, err = BytesToResultMeta(sr.ResultHash, value)
				if err != nil {
					score = rm.Score
				}
			}
		}
		toSort = append(toSort, scoredResult{
			sr:    sr,
			score: score,
		})
	}
	sort.Slice(toSort, func(i, j int) bool {
		return toSort[i].score < toSort[j].score
	})

	results = make([]SearchResult, 0)
	for _, ssr := range toSort {
		results = append(results, ssr.sr)
	}
	return results
}

func (sdb *SearchDB) GetAllHashes() [][32]byte {
	// hashes := sdb.db.GetAllHashes()
	// hashes = append(hashes, sdb.cache.GetAllHashes()...)
	return nil
}

func (sdb *SearchDB) GetForSync(timestamp uint64, sizeLimit int) []SearchResult {
	results := make([]SearchResult, 0)

	count := 0

	for _, sr := range sdb.searchesCache {
		if count >= sizeLimit {
			return results
		}
		if sr.Timestamp >= timestamp {
			results = append(results, sr)
			count++
		}
	}

	iter := sdb.searchesDB.NewIterator(nil, nil)
	for iter.Next() {
		if count >= sizeLimit {
			return results
		}

		sr, err := BytesToSearchResult(util.SliceToArray32(iter.Key()), iter.Value())
		if err != nil {
			continue
		}
		if sr.Timestamp >= timestamp {
			results = append(results, sr)
			count++
		}
	}
	iter.Release()
	return results
}

func (sdb *SearchDB) SyncFrom(results []SearchResult) {
	max := sdb.maximumCacheSize / SEARCH_RESULT_BYTE_SIZE
	count := 0
	for _, sr := range results {
		if count >= max {
			count = 0
			sdb.Flush()
		}
		if sr.IsConsistent() {
			sdb.searchesCache[sr.ResultHash] = sr
		}
		count++
	}
	sdb.setUpdated()
}

func (sdb *SearchDB) GetMetadataOf(hashes [][32]byte) []ResultMeta {
	metas := make(map[[32]byte]ResultMeta, 0)
	for _, h := range hashes {
		meta, present := sdb.metasCache[h]
		if present {
			metas[meta.ResultHash] = meta
		}
	}
	for _, h := range hashes {
		bytez, err := sdb.searchesDB.Get(util.Array32ToSlice(h), nil)
		if err == nil {
			meta, err := BytesToResultMeta(h, bytez)
			if err == nil {
				metas[meta.ResultHash] = meta
			}
		}
	}
	return util.MapToSlice(metas)
}

func (sdb *SearchDB) GetMetadataForSync(ts uint64, sizeLimit int) []ResultMeta {
	metas := make([]ResultMeta, 0)
	count := 0
	for _, rm := range sdb.metasCache {
		if count >= sizeLimit {
			return metas
		}
		if rm.Timestamp >= ts {
			metas = append(metas, rm)
			count++
		}
	}

	iter := sdb.searchesDB.NewIterator(nil, nil)
	for iter.Next() {
		if count >= sizeLimit {
			return metas
		}

		rm, err := BytesToResultMeta(util.SliceToArray32(iter.Key()), iter.Value())
		if err != nil {
			continue
		}
		if rm.Timestamp >= ts {
			metas = append(metas, rm)
			count++
		}
	}
	iter.Release()
	return metas
}

func (sdb *SearchDB) SyncResultsMetadataFrom(metas []ResultMeta) {
	max := sdb.maximumCacheSize / RESULT_META_BYTE_SIZE
	count := 0
	for _, rm := range metas {
		if count >= max {
			count = 0
			sdb.Flush()
		}
		prev, present := sdb.metasCache[rm.ResultHash]
		if present && prev.Score > 0 {
			//prevents spam
			rm.Score = (prev.Score + rm.Score) / 2
		}
		sdb.metasCache[rm.ResultHash] = rm
		count++
	}
	sdb.setUpdated()
}

func (sdb *SearchDB) InsertResult(sr SearchResult) {
	if sr.IsConsistent() {
		sdb.searchesCache[sr.ResultHash] = sr
		sdb.metasCache[sr.ResultHash] = NewResultMeta(sr.ResultHash, sr.Timestamp, 0, NONE)
	}
	sdb.setUpdated()
}

func (sdb *SearchDB) UpdateResultScore(hash [32]byte, increment int) {
	rm := sdb.metasCache[hash]
	rm.Score += uint16(increment)
	rm.UpdateTime()
	sdb.metasCache[hash] = rm
	sdb.setUpdated()
}

func (sdb *SearchDB) CalculateCacheSize() int {
	return len(sdb.searchesCache)*SEARCH_RESULT_BYTE_SIZE + len(sdb.metasCache)*RESULT_META_BYTE_SIZE
}

func (sdb *SearchDB) setUpdated() {
	sdb.flushed = false
	searchesLastTime := sdb.GetLastCachedSearchResultTimestamp()
	if searchesLastTime > 0 && searchesLastTime >= sdb.LastTimestamp {
		sdb.LastTimestamp = searchesLastTime
	}
	metasLastTime := sdb.GetLastCachedResultMetaTimestamp()
	if metasLastTime > 0 && metasLastTime > sdb.LastMetaTimestamp {
		sdb.LastMetaTimestamp = metasLastTime
	}
	if sdb.CalculateCacheSize() > int(sdb.maximumCacheSize) {
		sdb.Flush()
	}
}

func (sdb *SearchDB) InvalidateResult(rHash [32]byte) {
	rm := sdb.metasCache[rHash]
	rm.Invalidated = INVALIDATED
	sdb.metasCache[rHash] = rm
}

func (sdb *SearchDB) deleteInvalidated() {
	iter := sdb.metasDB.NewIterator(nil, nil)
	for iter.Next() {
		rm, err := BytesToResultMeta(util.SliceToArray32(iter.Key()), iter.Value())
		if err != nil {
			continue
		}
		if rm.Invalidated == INVALIDATED {
			sdb.searchesDB.Delete(iter.Key(), nil)
			if err != nil {
				logging.LogTrace("Error deleting invalidated:", err.Error())
			}
		}
	}
	iter.Release()
}

func (sdb *SearchDB) Flush() {
	if sdb.flushed {
		return
	}

	// sdb.LastTimestamp = sdb.GetLastCachedSearchResultTimestamp()
	// sdb.LastMetaTimestamp = sdb.GetLastCachedResultMetaTimestamp()

	// Flush SearchResults
	for h, sr := range sdb.searchesCache {
		sdb.searchesDB.Put(util.Array32ToSlice(h), sr.ToBytes(), nil)
	}

	// Flush ResultMetas
	for h, rm := range sdb.metasCache {
		key := util.Array32ToSlice(h)
		value, err := sdb.metasDB.Get(key, nil)
		if err != nil {
			prev, err := BytesToResultMeta(h, value)
			if err != nil && prev.Score > 0 {
				//prevents spam
				rm.Score = (rm.Score + prev.Score) / 2
			}
		}
		sdb.metasDB.Put(key, rm.ToBytes(), nil)
	}

	sdb.deleteInvalidated()

	// Clear cache
	sdb.searchesCache = make(map[[32]byte]SearchResult, 0)
	sdb.metasCache = make(map[[32]byte]ResultMeta, 0)

	sdb.flushed = true
}

func (sdb *SearchDB) Close() {
	sdb.searchesDB.Close()
	sdb.metasDB.Close()
}

// For tests
func (sdb *SearchDB) GetSearchesCache() map[[32]byte]SearchResult {
	return sdb.searchesCache
}

func (sdb *SearchDB) GetMetasCache() map[[32]byte]ResultMeta {
	return sdb.metasCache
}

func (sdb *SearchDB) GetSearchesDB() *leveldb.DB {
	return sdb.searchesDB
}
