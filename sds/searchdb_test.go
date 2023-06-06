package sds_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/sniffdogsniff/sds"
	"github.com/sniffdogsniff/util"
	"github.com/syndtr/goleveldb/leveldb"
)

const TEST_DIR = "./test_dir"

func setupDB() sds.SearchDB {
	// logging.InitLogging(logging.TRACE)
	db := sds.SearchDB{}
	os.Mkdir(TEST_DIR, 0707)
	os.Chmod(TEST_DIR+"/*", 0707)
	db.Open(TEST_DIR, 1024*1024*256)
	return db
}

func teardownDB() {
	err := os.RemoveAll(TEST_DIR)
	if err != nil {
		fmt.Println(err.Error())
	}
}

func differentValues(name string, a, b interface{}, t *testing.T) {
	t.Fatal("Different", name, "values: one id", b, "other is", a)
}

func assertMetaRecord(meta sds.ResultMeta, rHash [32]byte, score int, inv sds.Invalidation, t *testing.T) {
	if meta.ResultHash != rHash {
		differentValues("hash", meta.ResultHash, rHash, t)
	}
	if meta.Score != uint16(score) {
		differentValues("score", meta.Score, score, t)
	}
	if meta.Invalidated != inv {
		t.Fatalf("invalidated is different!")
	}
}

func assertSearchResult(sr sds.SearchResult, title, url, desc string, t *testing.T) {
	if sr.Title != title {
		differentValues("title", sr.Title, title, t)
	}
	if sr.Url != url {
		differentValues("url", sr.Url, url, t)
	}
	if sr.Description != desc {
		differentValues("timestamp", sr.Description, desc, t)
	}
}

func getAllDBSearchesAsMap(db *leveldb.DB) map[[32]byte]sds.SearchResult {
	searches := make(map[[32]byte]sds.SearchResult, 0)
	iter := db.NewIterator(nil, nil)
	for iter.Next() {
		sr, err := sds.BytesToSearchResult(util.SliceToArray32(iter.Key()), iter.Value())
		if err != nil {
			continue
		}
		searches[sr.ResultHash] = sr
	}
	return searches
}

func TestSearchResult_TOBYTES_FROMBYTES(t *testing.T) {
	one := sds.NewSearchResult("title1", "http://url1.net", "one")
	from, err := sds.BytesToSearchResult(one.ResultHash, one.ToBytes())

	if err != nil {
		t.Fail()
	}

	if one.ResultHash != from.ResultHash {
		differentValues("ResultHash", one.ResultHash, from.ResultHash, t)
	}
	if one.Timestamp != from.Timestamp {
		differentValues("Timestamp", one.Timestamp, from.Timestamp, t)
	}
	if one.Title != from.Title {
		differentValues("Title", one.Title, from.Title, t)
	}
	if one.Url != from.Url {
		differentValues("Url", one.Url, from.Url, t)
	}
	if one.Description != from.Description {
		differentValues("Title", one.Description, from.Description, t)
	}
}

func TestResultMeta_TOBYTES_FROMBYTES(t *testing.T) {
	one := sds.NewResultMeta(sds.NewSearchResult("title1", "http://url1.net", "one").ResultHash, 744, 234, sds.PENDING)
	from, err := sds.BytesToResultMeta(one.ResultHash, one.ToBytes())

	if err != nil {
		t.Fail()
	}

	if one.ResultHash != from.ResultHash {
		differentValues("ResultHash", one.ResultHash, from.ResultHash, t)
	}
	if one.Timestamp != from.Timestamp {
		differentValues("Timestamp", one.Timestamp, from.Timestamp, t)
	}
	if one.Score != from.Score {
		differentValues("Score", one.Score, from.Score, t)
	}
	if one.Invalidated != from.Invalidated {
		differentValues("Invalidated", one.Invalidated, from.Invalidated, t)
	}
}

func TestDeleteInvalidated(t *testing.T) {
	db := setupDB()
	one := sds.NewSearchResult("title1", "http://url1.net", "one")
	db.InsertResult(one)
	two := sds.NewSearchResult("title2", "http://url2.net", "two")
	db.InsertResult(two)
	three := sds.NewSearchResult("title3", "http://url3.net", "three")
	db.InsertResult(three)

	metas1 := make([]sds.ResultMeta, 0)
	metas1 = append(metas1, sds.NewResultMeta(one.ResultHash, 0, 12, sds.INVALIDATED))
	metas1[0].UpdateTime()
	metas1 = append(metas1, sds.NewResultMeta(two.ResultHash, 0, 23, sds.INVALIDATED))
	metas1[1].UpdateTime()

	db.SyncResultsMetadataFrom(metas1)
	db.Flush()

	all := getAllDBSearchesAsMap(db.GetSearchesDB())
	if len(all) != 1 {
		t.Fatalf("DB size wrong: %d", len(all))
	}
	if all[three.ResultHash] != three {
		t.Fatalf("want three eq three")
	}

	teardownDB()
}

func TestTimeBasedSync_Searches(t *testing.T) {
	db := setupDB()
	one := sds.NewSearchResult("title1", "http://url1.net", "one")
	db.InsertResult(one)
	db.InsertResult(sds.NewSearchResult("title2", "http://url2.net", "two"))
	db.InsertResult(sds.NewSearchResult("title3", "http://url3.net", "three"))

	time.Sleep(2 * time.Second)

	toSync := make([]sds.SearchResult, 0)
	// a duplicated entry with different timestamp
	toSync = append(toSync, sds.NewSearchResult("title1", "http://url1.net", "one"))
	toSync = append(toSync, sds.NewSearchResult("title4", "http://url4.net", "four"))
	toSync = append(toSync, sds.NewSearchResult("title5", "http://url5.net", "five"))

	db.SyncFrom(toSync)

	dbLen := len(db.GetSearchesCache())
	if !(dbLen == 5) {
		t.Fatalf("lenght of db != 5, actual: %d", dbLen)
	}

	teardownDB()
}

func TestFlush_TimestampNeverZero(t *testing.T) {
	db := setupDB()
	one := sds.NewSearchResult("title1", "http://url1.net", "one")
	db.InsertResult(one)
	db.InsertResult(sds.NewSearchResult("title2", "http://url2.net", "two"))
	db.InsertResult(sds.NewSearchResult("title3", "http://url3.net", "three"))

	time.Sleep(2 * time.Second)

	toSync := make([]sds.SearchResult, 0)
	// a duplicated entry with different timestamp
	toSync = append(toSync, sds.NewSearchResult("title1", "http://url1.net", "one"))
	toSync = append(toSync, sds.NewSearchResult("title4", "http://url4.net", "four"))
	toSync = append(toSync, sds.NewSearchResult("title5", "http://url5.net", "five"))

	db.SyncFrom(toSync)
	lastTs := db.GetLastCachedSearchResultTimestamp()
	if lastTs == 0 {
		t.Fatalf("last timestamp is 0")
	}

	db.Flush()
	if db.LastTimestamp == 0 {
		t.Fatalf("last timestamp is 0")
	}

	teardownDB()
}

func TestTimeBasedSync_Meta(t *testing.T) {
	db := setupDB()
	one := sds.NewSearchResult("title1", "http://url1.net", "one")
	db.InsertResult(one)

	metas1 := make([]sds.ResultMeta, 0)
	metas1 = append(metas1, sds.NewResultMeta(one.ResultHash, one.Timestamp+340, 12, 25))

	db.SyncResultsMetadataFrom(metas1)

	db.SyncResultsMetadataFrom([]sds.ResultMeta{sds.NewResultMeta(one.ResultHash, one.Timestamp+790, 34, 21)})

	metasDB := db.GetMetasCache()
	assertMetaRecord(metasDB[one.ResultHash], one.ResultHash, (12+34)/2, 21, t)

	teardownDB()
}

func TestDoSearch(t *testing.T) {
	db := setupDB()

	for i := 1; i < 10; i++ {
		sr := sds.NewSearchResult(fmt.Sprintf("test%d", i), "", fmt.Sprintf("description_test%d", i))
		db.InsertResult(sr)
		db.UpdateResultScore(sr.ResultHash, 20-i)
	}

	results := db.DoSearch("test")
	assertSearchResult(results[0], "test9", "", "description_test9", t)
	assertSearchResult(results[1], "test8", "", "description_test8", t)
	assertSearchResult(results[2], "test7", "", "description_test7", t)
	assertSearchResult(results[3], "test6", "", "description_test6", t)
	assertSearchResult(results[4], "test5", "", "description_test5", t)
	assertSearchResult(results[5], "test4", "", "description_test4", t)
	assertSearchResult(results[6], "test3", "", "description_test3", t)
	assertSearchResult(results[7], "test2", "", "description_test2", t)
	assertSearchResult(results[8], "test1", "", "description_test1", t)

	teardownDB()
}
