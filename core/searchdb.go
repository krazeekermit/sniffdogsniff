package core

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/sniffdogsniff/kademlia"
	"github.com/sniffdogsniff/logging"
	"github.com/sniffdogsniff/util"
	"github.com/syndtr/goleveldb/leveldb"
)

type Hash256 [32]byte

func SliceToHas256(data []byte) Hash256 {
	var bytes Hash256
	for i, b := range data {
		if i > 31 {
			break
		}
		bytes[i] = b
	}
	return bytes
}

func Hash256ToSlice(data Hash256) []byte {
	slice := make([]byte, 0)
	for _, b := range data {
		slice = append(slice, b)
	}
	return slice
}

func HashToB64UrlsafeString(hash Hash256) string {
	return base64.URLEncoding.EncodeToString(hash[:])
}

func B64UrlsafeStringToHash(b64 string) Hash256 {
	bytes, _ := base64.URLEncoding.DecodeString(b64)
	return SliceToHas256(bytes)
}

const SEARCHES_DB_FILE_NAME = "searches.db"
const METASS_DB_FILE_NAME = "searches_meta.db"

const SEARCH_RESULT_BYTE_SIZE = 768 // bytes
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
	ResultHash   Hash256
	QueryMetrics []kademlia.KadId
	Timestamp    uint64
	Title        string
	Url          string
	Properties   ResultPropertiesMap
	DataType     ResultDataType
}

/*
SearchResult Structure
[[HASH (32)][KadIds (80)][TIMESTAMP (8)][TITLE][URL][PROPERTIES]] = 768 bytes
*/
func NewSearchResult(title, url string, properties ResultPropertiesMap, dataType ResultDataType) SearchResult {
	const FIXED_LENGHT = 32 + 1 + 80 + 8 + 1 + 1 + 1 + 1 // Hash=32, time=8, len(title)=1, len(url)=1, len(desc)=1, datatype=1
	// SHRINK TITLE AND PROPERTIES TO FIT 768 byte max size

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

	var metrics []kademlia.KadId
	description, ok := properties[RP_DESCRIPTION]
	if ok {
		metrics = EvalQueryMetrics(fmt.Sprintf("%s %s", title, description))
	} else {
		metrics = EvalQueryMetrics(title)
	}

	rs := SearchResult{
		QueryMetrics: metrics,
		Timestamp:    uint64(util.CurrentUnixTime()),
		Title:        title,
		Url:          url,
		DataType:     dataType,
		Properties:   properties,
	}
	rs.ReHash()
	return rs
}

func BytesToSearchResult(hash Hash256, bytez []byte) (SearchResult, error) {
	buf := bytes.NewBuffer(bytez)

	metricsLen, err := buf.ReadByte() // TITLE
	if err != nil {
		return SearchResult{}, err
	}
	metrics := make([]kademlia.KadId, metricsLen)
	for i := 0; i < int(metricsLen); i++ {
		mBytez := make([]byte, 20)
		n, err := buf.Read(mBytez)
		if err != nil || n != 20 {
			return SearchResult{}, err
		}
		metrics[i] = kademlia.KadIdFromBytes(mBytez)
	}

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
		ResultHash:   hash,
		QueryMetrics: metrics,
		Timestamp:    binary.LittleEndian.Uint64(bts),
		Title:        string(btitle),
		Url:          string(burl),
		Properties:   properties,
		DataType:     ResultDataType(dataType),
	}, nil
}

func (sr *SearchResult) ToBytes() []byte {
	buf := bytes.NewBuffer(nil)
	buf.WriteByte(byte(len(sr.QueryMetrics)))
	for _, mid := range sr.QueryMetrics {
		buf.Write(mid.ToBytes())
	}
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

func (sr *SearchResult) calculateHash() Hash256 {
	thBytes := make([]byte, 0)
	for _, m := range sr.QueryMetrics {
		thBytes = append(thBytes, m.ToBytes()...)
	}
	thBytes = append(thBytes, []byte(sr.Title)...)
	thBytes = append(thBytes, []byte(sr.Url)...)
	for k, p := range sr.Properties {
		thBytes = append(thBytes, k)
		thBytes = append(thBytes, []byte(p)...)
	}
	thBytes = append(thBytes, byte(sr.DataType))

	return sha256.Sum256(thBytes)
}

func (sr *SearchResult) CheckIntegrity() bool {
	return sr.ResultHash == sr.calculateHash()
}

func (sr *SearchResult) ReHash() {
	sr.ResultHash = sr.calculateHash()
}

func (sr *SearchResult) HashAsB64UrlSafeStr() string {
	return HashToB64UrlsafeString(sr.ResultHash)
}

func (sr *SearchResult) SafeGetProperty(propKey uint8) string {
	property, ok := sr.Properties[propKey]
	if ok {
		return property
	}
	return ""
}

func (sr *SearchResult) String() string {
	return fmt.Sprintf("(Hash: %s, Title: %s, Url %s)", fmt.Sprint(sr.ResultHash), sr.Title, sr.Url)
}

/****************************************************************************************************/

/**
* SearchDB is containing the db (actual db on disk) and cache (a cached db stored in ram)
* Sync operations GetResults Sync Results ecct are expensive in terms of time because they
* read and write to the disk, so we decided to add an in-memory database to store the data temporarly
* and then flush to disk when the cache size reaches its limit
**/

type SearchDB struct {
	expirationThr    uint64
	maximumCacheSize int

	searchesDB           *leveldb.DB
	searchesCache        map[Hash256]SearchResult
	lastPublishTimeCache map[Hash256]int64
	flushed              bool
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

func (sdb *SearchDB) Open(workDir string, maxCacheSize int, expirationThr uint64) {
	sdb.maximumCacheSize = maxCacheSize
	sdb.expirationThr = expirationThr
	logging.LogTrace("DEBUG: SearchDB started with", maxCacheSize, "Bytes of cache")

	sdb.searchesDB = openDB(filepath.Join(workDir, SEARCHES_DB_FILE_NAME))

	// Initialize cache
	sdb.searchesCache = make(map[Hash256]SearchResult, 0)
	sdb.lastPublishTimeCache = make(map[Hash256]int64)
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

	for _, sr := range sdb.searchesCache {
		if matchesSearch(text, sr) {
			results = append(results, sr)
		}
	}

	iter := sdb.searchesDB.NewIterator(nil, nil)
	for iter.Next() {
		sr, err := BytesToSearchResult(SliceToHas256(iter.Key()), iter.Value())
		if err != nil {
			continue
		}
		if matchesSearch(text, sr) {
			results = append(results, sr)
		}
	}
	iter.Release()
	return results
}

func (sdb *SearchDB) ResultsToPublish() []SearchResult {
	//first delete expired so are not going to be republished
	sdb.deleteExpiredEnries()

	results := make([]SearchResult, 0)

	timeNow := util.CurrentUnixTime()
	for _, sr := range sdb.searchesCache {
		if timeNow-sdb.lastPublishTimeCache[sr.ResultHash] >= util.TIME_HOUR_UNIX {
			sdb.lastPublishTimeCache[sr.ResultHash] = timeNow
			results = append(results, sr)
		}
	}

	iter := sdb.searchesDB.NewIterator(nil, nil)
	for iter.Next() {
		sr, err := BytesToSearchResult(SliceToHas256(iter.Key()), iter.Value())
		if err != nil {
			continue
		}
		_, present := sdb.lastPublishTimeCache[sr.ResultHash]
		if !present {
			sdb.lastPublishTimeCache[sr.ResultHash] = 0
		}
		if timeNow-sdb.lastPublishTimeCache[sr.ResultHash] >= util.TIME_HOUR_UNIX {
			sdb.lastPublishTimeCache[sr.ResultHash] = timeNow
			results = append(results, sr)
		}
	}
	iter.Release()

	return results
}

func (sdb *SearchDB) InsertResult(sr SearchResult) {
	if sr.CheckIntegrity() {
		timeNow := util.CurrentUnixTime()
		sdb.lastPublishTimeCache[sr.ResultHash] = timeNow

		sr.Timestamp = uint64(timeNow)
		sdb.searchesCache[sr.ResultHash] = sr
	}
	sdb.setUpdated()
}

func (sdb *SearchDB) InsertResults(results []SearchResult) {
	for _, sr := range results {
		sdb.InsertResult(sr)
	}
}

// func (sdb *SearchDB) UpdateResultScore(hash Hash256, increment int) {
// 	rm := sdb.metasCache[hash]
// 	rm.Score += uint16(increment)
// 	rm.UpdateTime()
// 	sdb.metasCache[hash] = rm
// 	sdb.setUpdated()
// }

func (sdb *SearchDB) CalculateCacheSize() int {
	return len(sdb.searchesCache) * SEARCH_RESULT_BYTE_SIZE
}

func (sdb *SearchDB) deleteExpiredEnries() {
	//expThrMinutes avoids deletions every second - for disk resource uses
	expThrMinutes := sdb.expirationThr / 60

	toKeep := make([]SearchResult, 0)

	timeNow := util.CurrentUnixTime()
	for _, sr := range sdb.searchesCache {
		if ((uint64(timeNow) - sr.Timestamp) / 60) < expThrMinutes {
			toKeep = append(toKeep, sr)
		}
	}

	sdb.searchesCache = make(map[Hash256]SearchResult)
	for _, v := range toKeep {
		sdb.searchesCache[v.ResultHash] = v
	}

	toDelete := make([]SearchResult, 0)
	iter := sdb.searchesDB.NewIterator(nil, nil)
	for iter.Next() {
		sr, err := BytesToSearchResult(SliceToHas256(iter.Key()), iter.Value())
		if err != nil {
			continue
		}
		if ((uint64(timeNow) - sr.Timestamp) / 60) >= expThrMinutes {
			toDelete = append(toDelete, sr)
		}
	}
	iter.Release()

	for _, v := range toDelete {
		sdb.searchesDB.Delete(Hash256ToSlice(v.ResultHash), nil)
	}
}

func (sdb *SearchDB) setUpdated() {
	sdb.flushed = false
	if sdb.CalculateCacheSize() > int(sdb.maximumCacheSize) {
		sdb.Flush()
	}

	sdb.deleteExpiredEnries()
}

func (sdb *SearchDB) Flush() {
	if sdb.flushed {
		return
	}

	// sdb.LastTimestamp = sdb.GetLastCachedSearchResultTimestamp()
	// sdb.LastMetaTimestamp = sdb.GetLastCachedResultMetaTimestamp()

	// Flush SearchResults
	for h, sr := range sdb.searchesCache {
		sdb.searchesDB.Put(Hash256ToSlice(h), sr.ToBytes(), nil)
	}

	// Clear cache
	sdb.searchesCache = make(map[Hash256]SearchResult, 0)

	sdb.flushed = true
}

func (sdb *SearchDB) Close() {
	sdb.searchesDB.Close()
}

// For tests
func (sdb *SearchDB) GetSearchesCache() map[Hash256]SearchResult {
	return sdb.searchesCache
}

func (sdb *SearchDB) GetSearchesDB() *leveldb.DB {
	return sdb.searchesDB
}
