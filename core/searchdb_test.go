package core_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/sniffdogsniff/core"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vmihailenco/msgpack/v5"
)

const TEST_DIR = "./test_dir"

func setupDB() core.SearchDB {
	// logging.InitLogging(logging.TRACE)
	db := core.SearchDB{}
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

func assertMetaRecord(meta core.ResultMeta, rHash core.Hash256, score uint16, inv int8, t *testing.T) {
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

func assertMetaEq(meta, meta0 core.ResultMeta, t *testing.T) {
	assertMetaRecord(meta, meta0.ResultHash, meta0.Score, meta0.Invalidated, t)
}

func assertSearchResult(sr core.SearchResult, rHash core.Hash256, title, url string, properties core.ResultPropertiesMap, t *testing.T) {
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

func assertSearchResultEq(sr, sr0 core.SearchResult, t *testing.T) {
	assertSearchResult(sr, sr0.ResultHash, sr0.Title, sr0.Url, sr0.Properties, t)
}

func getAllDBSearchesAsMap(db *leveldb.DB) map[core.Hash256]core.SearchResult {
	searches := make(map[core.Hash256]core.SearchResult, 0)
	iter := db.NewIterator(nil, nil)
	for iter.Next() {
		sr, err := core.BytesToSearchResult(core.SliceToHas256(iter.Key()), iter.Value())
		if err != nil {
			continue
		}
		searches[sr.ResultHash] = sr
	}
	return searches
}

func TestSearchResult_TOBYTES_FROMBYTES(t *testing.T) {
	one := core.NewSearchResult("title1", "http://url1.net", core.ResultPropertiesMap{
		core.RP_DESCRIPTION: "descriptionnnnnnn",
		core.RP_THUMB_LINK:  "http://blabla",
	}, core.IMAGE_DATA_TYPE)

	from, err := core.BytesToSearchResult(one.ResultHash, one.ToBytes())

	if err != nil {
		t.Fail()
	}

	assertSearchResult(one, from.ResultHash, from.Title, from.Url, from.Properties, t)
}

func TestSearchResult_MarshalUnmarshal(t *testing.T) {
	one := core.NewSearchResult("title1", "http://url1.net", core.ResultPropertiesMap{
		core.RP_DESCRIPTION: "descriptionnnnnnn",
		core.RP_THUMB_LINK:  "http://blabla",
	}, core.IMAGE_DATA_TYPE)

	b_one, err := msgpack.Marshal(one)
	if err != nil {
		t.Fail()
	}

	var from core.SearchResult
	err = msgpack.Unmarshal(b_one, &from)
	if err != nil {
		t.Fail()
	}

	assertSearchResult(one, from.ResultHash, from.Title, from.Url, from.Properties, t)
}

func TestResultMeta_TOBYTES_FROMBYTES(t *testing.T) {
	one := core.NewResultMeta(core.NewSearchResult("title1", "http://url1.net",
		core.ResultPropertiesMap{}, core.VIDEO_DATA_TYPE).ResultHash, 744, 234, 5)
	from, err := core.BytesToResultMeta(one.ResultHash, one.ToBytes())

	if err != nil {
		t.Fail()
	}

	assertMetaRecord(one, from.ResultHash, from.Score, from.Invalidated, t)
}

func TestResultMeta_MarshalUnmarshal(t *testing.T) {
	one := core.NewResultMeta(core.NewSearchResult("title1", "http://url1.net",
		core.ResultPropertiesMap{}, core.VIDEO_DATA_TYPE).ResultHash, 744, 234, 5)

	b_one, err := msgpack.Marshal(one)
	if err != nil {
		t.Fail()
	}

	var from core.ResultMeta
	err = msgpack.Unmarshal(b_one, &from)
	if err != nil {
		t.Fail()
	}

	assertMetaRecord(one, from.ResultHash, from.Score, from.Invalidated, t)
}

func TestDeleteInvalidated(t *testing.T) {
	db := setupDB()
	one := core.NewSearchResult("title1", "http://url1.net",
		core.ResultPropertiesMap{core.RP_DESCRIPTION: "one"}, core.LINK_DATA_TYPE)
	db.InsertResult(one)
	two := core.NewSearchResult("title2", "http://url2.net",
		core.ResultPropertiesMap{core.RP_DESCRIPTION: "two"}, core.LINK_DATA_TYPE)
	db.InsertResult(two)
	three := core.NewSearchResult("title3", "http://url3.net",
		core.ResultPropertiesMap{core.RP_DESCRIPTION: "three"}, core.LINK_DATA_TYPE)
	db.InsertResult(three)

	metas1 := make([]core.ResultMeta, 0)
	metas1 = append(metas1, core.NewResultMeta(one.ResultHash, 0, 12, core.INVALIDATION_LEVEL_INVALIDATED))
	metas1[0].UpdateTime()
	metas1 = append(metas1, core.NewResultMeta(two.ResultHash, 0, 23, core.INVALIDATION_LEVEL_INVALIDATED))
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
	one := core.NewSearchResult("title1", "http://url1.net",
		core.ResultPropertiesMap{core.RP_DESCRIPTION: "one"}, core.LINK_DATA_TYPE)
	db.InsertResult(one)
	db.InsertResult(core.NewSearchResult("title2", "http://url2.net",
		core.ResultPropertiesMap{core.RP_DESCRIPTION: "two"}, core.LINK_DATA_TYPE))
	db.InsertResult(core.NewSearchResult("title3", "http://url3.net",
		core.ResultPropertiesMap{core.RP_DESCRIPTION: "three"}, core.LINK_DATA_TYPE))

	time.Sleep(2 * time.Second)

	toSync := make([]core.SearchResult, 0)
	// a duplicated entry with different timestamp
	toSync = append(toSync, core.NewSearchResult("title1", "http://url1.net",
		core.ResultPropertiesMap{core.RP_DESCRIPTION: "one"}, core.LINK_DATA_TYPE))
	toSync = append(toSync, core.NewSearchResult("title4", "http://url4.net",
		core.ResultPropertiesMap{core.RP_DESCRIPTION: "four"}, core.LINK_DATA_TYPE))
	toSync = append(toSync, core.NewSearchResult("title5", "http://url5.net",
		core.ResultPropertiesMap{core.RP_DESCRIPTION: "five"}, core.LINK_DATA_TYPE))

	db.SyncFrom(toSync)

	dbLen := len(db.GetSearchesCache())
	if !(dbLen == 5) {
		t.Fatalf("lenght of db != 5, actual: %d", dbLen)
	}

	teardownDB()
}

func TestFlush_TimestampNeverZero(t *testing.T) {
	db := setupDB()
	one := core.NewSearchResult("title1", "http://url1.net",
		core.ResultPropertiesMap{core.RP_DESCRIPTION: "one"}, core.LINK_DATA_TYPE)
	db.InsertResult(one)
	db.InsertResult(core.NewSearchResult("title2", "http://url2.net",
		core.ResultPropertiesMap{core.RP_DESCRIPTION: "two"}, core.LINK_DATA_TYPE))
	db.InsertResult(core.NewSearchResult("title3", "http://url3.net",
		core.ResultPropertiesMap{core.RP_DESCRIPTION: "three"}, core.LINK_DATA_TYPE))

	time.Sleep(2 * time.Second)

	toSync := make([]core.SearchResult, 0)
	// a duplicated entry with different timestamp
	toSync = append(toSync, core.NewSearchResult("title1", "http://url1.net",
		core.ResultPropertiesMap{core.RP_DESCRIPTION: "one"}, core.LINK_DATA_TYPE))
	toSync = append(toSync, core.NewSearchResult("title4", "http://url4.net",
		core.ResultPropertiesMap{core.RP_DESCRIPTION: "four"}, core.LINK_DATA_TYPE))
	toSync = append(toSync, core.NewSearchResult("title5", "http://url5.net",
		core.ResultPropertiesMap{core.RP_DESCRIPTION: "five"}, core.LINK_DATA_TYPE))

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
	one := core.NewSearchResult("title1", "http://url1.net",
		core.ResultPropertiesMap{core.RP_DESCRIPTION: "one"}, core.LINK_DATA_TYPE)
	db.InsertResult(one)

	metas1 := make([]core.ResultMeta, 0)
	metas1 = append(metas1, core.NewResultMeta(one.ResultHash, one.Timestamp+340, 12, 25))

	db.SyncResultsMetadataFrom(metas1)

	db.SyncResultsMetadataFrom([]core.ResultMeta{core.NewResultMeta(one.ResultHash, one.Timestamp+790, 34, 21)})

	metasDB := db.GetMetasCache()
	assertMetaRecord(metasDB[one.ResultHash], one.ResultHash, (12+34)/2, 21, t)

	teardownDB()
}

func TestDoSearch(t *testing.T) {
	db := setupDB()

	for i := 1; i < 10; i++ {
		sr := core.NewSearchResult(fmt.Sprintf("test%d", i), "",
			core.ResultPropertiesMap{core.RP_DESCRIPTION: fmt.Sprintf("description_test%d", i)},
			core.LINK_DATA_TYPE)
		db.InsertResult(sr)
		db.UpdateResultScore(sr.ResultHash, 20-i)
	}

	results := db.DoSearch("test")
	assertSearchResult(results[0], results[0].ResultHash, "test9", "",
		core.ResultPropertiesMap{core.RP_DESCRIPTION: "description_test9"}, t)
	assertSearchResult(results[1], results[1].ResultHash, "test8", "",
		core.ResultPropertiesMap{core.RP_DESCRIPTION: "description_test8"}, t)
	assertSearchResult(results[2], results[2].ResultHash, "test7", "",
		core.ResultPropertiesMap{core.RP_DESCRIPTION: "description_test7"}, t)
	assertSearchResult(results[3], results[3].ResultHash, "test6", "",
		core.ResultPropertiesMap{core.RP_DESCRIPTION: "description_test6"}, t)
	assertSearchResult(results[4], results[4].ResultHash, "test5", "",
		core.ResultPropertiesMap{core.RP_DESCRIPTION: "description_test5"}, t)
	assertSearchResult(results[5], results[5].ResultHash, "test4", "",
		core.ResultPropertiesMap{core.RP_DESCRIPTION: "description_test4"}, t)
	assertSearchResult(results[6], results[6].ResultHash, "test3", "",
		core.ResultPropertiesMap{core.RP_DESCRIPTION: "description_test3"}, t)
	assertSearchResult(results[7], results[7].ResultHash, "test2", "",
		core.ResultPropertiesMap{core.RP_DESCRIPTION: "description_test2"}, t)
	assertSearchResult(results[8], results[8].ResultHash, "test1", "",
		core.ResultPropertiesMap{core.RP_DESCRIPTION: "description_test1"}, t)

	teardownDB()
}
