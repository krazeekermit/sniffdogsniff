package sds_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/sniffdogsniff/sds"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vmihailenco/msgpack"
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

func assertMetaRecord(meta sds.ResultMeta, rHash sds.Hash256, score uint16, inv int8, t *testing.T) {
	if meta.ResultHash != rHash {
		differentValues("hash", meta.ResultHash, rHash, t)
	}
	if meta.Score != score {
		differentValues("score", meta.Score, score, t)
	}
	if meta.Invalidated != inv {
		t.Fatalf("invalidated is different!")
	}
}

func assertSearchResult(sr sds.SearchResult, rHash sds.Hash256, title, url string, properties sds.ResultPropertiesMap, t *testing.T) {
	if sr.ResultHash != rHash {
		differentValues("hash", sr.ResultHash, rHash, t)
	}
	if sr.Title != title {
		differentValues("title", sr.Title, title, t)
	}
	if sr.Url != url {
		differentValues("url", sr.Url, url, t)
	}
	for k, p := range properties {
		p1 := sr.SafeGetProperty(k)
		if p1 != p {
			differentValues(fmt.Sprintf("property[%d]", k), p, p1, t)
		}
	}
}

func getAllDBSearchesAsMap(db *leveldb.DB) map[sds.Hash256]sds.SearchResult {
	searches := make(map[sds.Hash256]sds.SearchResult, 0)
	iter := db.NewIterator(nil, nil)
	for iter.Next() {
		sr, err := sds.BytesToSearchResult(sds.SliceToHas256(iter.Key()), iter.Value())
		if err != nil {
			continue
		}
		searches[sr.ResultHash] = sr
	}
	return searches
}

func TestSearchResult_TOBYTES_FROMBYTES(t *testing.T) {
	one := sds.NewSearchResult("title1", "http://url1.net", sds.ResultPropertiesMap{
		sds.RP_DESCRIPTION: "descriptionnnnnnn",
		sds.RP_THUMB_LINK:  "http://blabla",
	}, sds.IMAGE_DATA_TYPE)

	from, err := sds.BytesToSearchResult(one.ResultHash, one.ToBytes())

	if err != nil {
		t.Fail()
	}

	assertSearchResult(one, from.ResultHash, from.Title, from.Url, from.Properties, t)
}

func TestSearchResult_SERIALIZE_MSGPACK(t *testing.T) {
	one := sds.NewSearchResult("title1", "http://url1.net", sds.ResultPropertiesMap{
		sds.RP_DESCRIPTION: "descriptionnnnnnn",
		sds.RP_THUMB_LINK:  "http://blabla",
	}, sds.IMAGE_DATA_TYPE)

	b_one, err := msgpack.Marshal(one)
	if err != nil {
		t.Fail()
	}

	var from sds.SearchResult
	err = msgpack.Unmarshal(b_one, &from)
	if err != nil {
		t.Fail()
	}

	assertSearchResult(one, from.ResultHash, from.Title, from.Url, from.Properties, t)
}

func TestResultMeta_TOBYTES_FROMBYTES(t *testing.T) {
	one := sds.NewResultMeta(sds.NewSearchResult("title1", "http://url1.net",
		sds.ResultPropertiesMap{}, sds.VIDEO_DATA_TYPE).ResultHash, 744, 234, 5)
	from, err := sds.BytesToResultMeta(one.ResultHash, one.ToBytes())

	if err != nil {
		t.Fail()
	}

	assertMetaRecord(one, from.ResultHash, from.Score, from.Invalidated, t)
}

func TestResultMeta_SERIALIZE_MSGPACK(t *testing.T) {
	one := sds.NewResultMeta(sds.NewSearchResult("title1", "http://url1.net",
		sds.ResultPropertiesMap{}, sds.VIDEO_DATA_TYPE).ResultHash, 744, 234, 5)

	b_one, err := msgpack.Marshal(one)
	if err != nil {
		t.Fail()
	}

	var from sds.ResultMeta
	err = msgpack.Unmarshal(b_one, &from)
	if err != nil {
		t.Fail()
	}

	assertMetaRecord(one, from.ResultHash, from.Score, from.Invalidated, t)
}

func TestDeleteInvalidated(t *testing.T) {
	db := setupDB()
	one := sds.NewSearchResult("title1", "http://url1.net",
		sds.ResultPropertiesMap{sds.RP_DESCRIPTION: "one"}, sds.LINK_DATA_TYPE)
	db.InsertResult(one)
	two := sds.NewSearchResult("title2", "http://url2.net",
		sds.ResultPropertiesMap{sds.RP_DESCRIPTION: "two"}, sds.LINK_DATA_TYPE)
	db.InsertResult(two)
	three := sds.NewSearchResult("title3", "http://url3.net",
		sds.ResultPropertiesMap{sds.RP_DESCRIPTION: "three"}, sds.LINK_DATA_TYPE)
	db.InsertResult(three)

	metas1 := make([]sds.ResultMeta, 0)
	metas1 = append(metas1, sds.NewResultMeta(one.ResultHash, 0, 12, sds.INVALIDATION_LEVEL_INVALIDATED))
	metas1[0].UpdateTime()
	metas1 = append(metas1, sds.NewResultMeta(two.ResultHash, 0, 23, sds.INVALIDATION_LEVEL_INVALIDATED))
	metas1[1].UpdateTime()

	db.SyncResultsMetadataFrom(metas1)
	db.Flush()

	all := getAllDBSearchesAsMap(db.GetSearchesDB())
	if len(all) != 1 {
		t.Fatalf("DB size wrong: %d", len(all))
	}

	assertSearchResult(all[three.ResultHash], three.ResultHash, three.Title, three.Url, three.Properties, t)

	teardownDB()
}

func TestTimeBasedSync_Searches(t *testing.T) {
	db := setupDB()
	one := sds.NewSearchResult("title1", "http://url1.net",
		sds.ResultPropertiesMap{sds.RP_DESCRIPTION: "one"}, sds.LINK_DATA_TYPE)
	db.InsertResult(one)
	db.InsertResult(sds.NewSearchResult("title2", "http://url2.net",
		sds.ResultPropertiesMap{sds.RP_DESCRIPTION: "two"}, sds.LINK_DATA_TYPE))
	db.InsertResult(sds.NewSearchResult("title3", "http://url3.net",
		sds.ResultPropertiesMap{sds.RP_DESCRIPTION: "three"}, sds.LINK_DATA_TYPE))

	time.Sleep(2 * time.Second)

	toSync := make([]sds.SearchResult, 0)
	// a duplicated entry with different timestamp
	toSync = append(toSync, sds.NewSearchResult("title1", "http://url1.net",
		sds.ResultPropertiesMap{sds.RP_DESCRIPTION: "one"}, sds.LINK_DATA_TYPE))
	toSync = append(toSync, sds.NewSearchResult("title4", "http://url4.net",
		sds.ResultPropertiesMap{sds.RP_DESCRIPTION: "four"}, sds.LINK_DATA_TYPE))
	toSync = append(toSync, sds.NewSearchResult("title5", "http://url5.net",
		sds.ResultPropertiesMap{sds.RP_DESCRIPTION: "five"}, sds.LINK_DATA_TYPE))

	db.SyncFrom(toSync)

	dbLen := len(db.GetSearchesCache())
	if !(dbLen == 5) {
		t.Fatalf("lenght of db != 5, actual: %d", dbLen)
	}

	teardownDB()
}

func TestFlush_TimestampNeverZero(t *testing.T) {
	db := setupDB()
	one := sds.NewSearchResult("title1", "http://url1.net",
		sds.ResultPropertiesMap{sds.RP_DESCRIPTION: "one"}, sds.LINK_DATA_TYPE)
	db.InsertResult(one)
	db.InsertResult(sds.NewSearchResult("title2", "http://url2.net",
		sds.ResultPropertiesMap{sds.RP_DESCRIPTION: "two"}, sds.LINK_DATA_TYPE))
	db.InsertResult(sds.NewSearchResult("title3", "http://url3.net",
		sds.ResultPropertiesMap{sds.RP_DESCRIPTION: "three"}, sds.LINK_DATA_TYPE))

	time.Sleep(2 * time.Second)

	toSync := make([]sds.SearchResult, 0)
	// a duplicated entry with different timestamp
	toSync = append(toSync, sds.NewSearchResult("title1", "http://url1.net",
		sds.ResultPropertiesMap{sds.RP_DESCRIPTION: "one"}, sds.LINK_DATA_TYPE))
	toSync = append(toSync, sds.NewSearchResult("title4", "http://url4.net",
		sds.ResultPropertiesMap{sds.RP_DESCRIPTION: "four"}, sds.LINK_DATA_TYPE))
	toSync = append(toSync, sds.NewSearchResult("title5", "http://url5.net",
		sds.ResultPropertiesMap{sds.RP_DESCRIPTION: "five"}, sds.LINK_DATA_TYPE))

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
	one := sds.NewSearchResult("title1", "http://url1.net",
		sds.ResultPropertiesMap{sds.RP_DESCRIPTION: "one"}, sds.LINK_DATA_TYPE)
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
		sr := sds.NewSearchResult(fmt.Sprintf("test%d", i), "",
			sds.ResultPropertiesMap{sds.RP_DESCRIPTION: fmt.Sprintf("description_test%d", i)},
			sds.LINK_DATA_TYPE)
		db.InsertResult(sr)
		db.UpdateResultScore(sr.ResultHash, 20-i)
	}

	results := db.DoSearch("test")
	assertSearchResult(results[0], results[0].ResultHash, "test9", "",
		sds.ResultPropertiesMap{sds.RP_DESCRIPTION: "description_test9"}, t)
	assertSearchResult(results[1], results[1].ResultHash, "test8", "",
		sds.ResultPropertiesMap{sds.RP_DESCRIPTION: "description_test8"}, t)
	assertSearchResult(results[2], results[2].ResultHash, "test7", "",
		sds.ResultPropertiesMap{sds.RP_DESCRIPTION: "description_test7"}, t)
	assertSearchResult(results[3], results[3].ResultHash, "test6", "",
		sds.ResultPropertiesMap{sds.RP_DESCRIPTION: "description_test6"}, t)
	assertSearchResult(results[4], results[4].ResultHash, "test5", "",
		sds.ResultPropertiesMap{sds.RP_DESCRIPTION: "description_test5"}, t)
	assertSearchResult(results[5], results[5].ResultHash, "test4", "",
		sds.ResultPropertiesMap{sds.RP_DESCRIPTION: "description_test4"}, t)
	assertSearchResult(results[6], results[6].ResultHash, "test3", "",
		sds.ResultPropertiesMap{sds.RP_DESCRIPTION: "description_test3"}, t)
	assertSearchResult(results[7], results[7].ResultHash, "test2", "",
		sds.ResultPropertiesMap{sds.RP_DESCRIPTION: "description_test2"}, t)
	assertSearchResult(results[8], results[8].ResultHash, "test1", "",
		sds.ResultPropertiesMap{sds.RP_DESCRIPTION: "description_test1"}, t)

	teardownDB()
}
