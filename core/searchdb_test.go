package core_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/sniffdogsniff/core"
	"github.com/sniffdogsniff/util"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/vmihailenco/msgpack/v5"
)

const TEST_DIR = "./test_dir"

func setupDB() core.SearchDB {
	// logging.InitLogging(logging.TRACE)
	db := core.SearchDB{}
	os.Mkdir(TEST_DIR, 0707)
	os.Chmod(TEST_DIR+"/*", 0707)
	db.Open(TEST_DIR, 1024*1024*256, 24*3600)
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

func Test_DeleteExpired_Flushed(t *testing.T) {
	timeNow := time.Now().Unix()
	util.SetTestTime(timeNow)

	db := setupDB()
	one := core.NewSearchResult("title1", "http://url1.net",
		core.ResultPropertiesMap{core.RP_DESCRIPTION: "one"}, core.LINK_DATA_TYPE)
	db.InsertResult(one)
	two := core.NewSearchResult("title2", "http://url2.net",
		core.ResultPropertiesMap{core.RP_DESCRIPTION: "two"}, core.LINK_DATA_TYPE)
	db.InsertResult(two)

	timeNow += 24 * 3601
	util.SetTestTime(timeNow)
	three := core.NewSearchResult("title3", "http://url3.net",
		core.ResultPropertiesMap{core.RP_DESCRIPTION: "three"}, core.LINK_DATA_TYPE)
	db.InsertResult(three)

	timeNow += 1801
	util.SetTestTime(timeNow)
	four := core.NewSearchResult("title4", "http://url4.net",
		core.ResultPropertiesMap{core.RP_DESCRIPTION: "four"}, core.LINK_DATA_TYPE)
	db.InsertResult(four)

	db.Flush()
	searches := getAllDBSearchesAsMap(db.GetSearchesDB())

	if len(searches) > 2 {
		t.Fatal()
	}

	// deleted after 24 hours
	_, present := searches[one.ResultHash]
	if present {
		t.Fatal()
	}
	_, present = searches[two.ResultHash]
	if present {
		t.Fatal()
	}

	// present
	_, present = searches[three.ResultHash]
	if !present {
		t.Fatal()
	}
	_, present = searches[four.ResultHash]
	if !present {
		t.Fatal()
	}

	teardownDB()
}

func Test_DeleteExpired_Cached(t *testing.T) {
	timeNow := time.Now().Unix()
	util.SetTestTime(timeNow)

	db := setupDB()
	one := core.NewSearchResult("title1", "http://url1.net",
		core.ResultPropertiesMap{core.RP_DESCRIPTION: "one"}, core.LINK_DATA_TYPE)
	db.InsertResult(one)
	two := core.NewSearchResult("title2", "http://url2.net",
		core.ResultPropertiesMap{core.RP_DESCRIPTION: "two"}, core.LINK_DATA_TYPE)
	db.InsertResult(two)

	timeNow += 24 * 3601
	util.SetTestTime(timeNow)
	three := core.NewSearchResult("title3", "http://url3.net",
		core.ResultPropertiesMap{core.RP_DESCRIPTION: "three"}, core.LINK_DATA_TYPE)
	db.InsertResult(three)

	timeNow += 1801
	util.SetTestTime(timeNow)
	four := core.NewSearchResult("title4", "http://url4.net",
		core.ResultPropertiesMap{core.RP_DESCRIPTION: "four"}, core.LINK_DATA_TYPE)
	db.InsertResult(four)

	if len(db.GetSearchesCache()) > 2 {
		t.Fatal()
	}

	// deleted after 24 hours
	_, present := db.GetSearchesCache()[one.ResultHash]
	if present {
		t.Fatal()
	}
	_, present = db.GetSearchesCache()[two.ResultHash]
	if present {
		t.Fatal()
	}

	// present
	_, present = db.GetSearchesCache()[three.ResultHash]
	if !present {
		t.Fatal()
	}
	_, present = db.GetSearchesCache()[four.ResultHash]
	if !present {
		t.Fatal()
	}

	teardownDB()
}

func Test_ResultsToRepublish(t *testing.T) {
	timeNow := time.Now().Unix()
	util.SetTestTime(timeNow)

	db := setupDB()
	one := core.NewSearchResult("title1", "http://url1.net",
		core.ResultPropertiesMap{core.RP_DESCRIPTION: "one"}, core.LINK_DATA_TYPE)
	db.InsertResult(one)

	timeNow += 1801
	util.SetTestTime(timeNow)
	two := core.NewSearchResult("title2", "http://url2.net",
		core.ResultPropertiesMap{core.RP_DESCRIPTION: "two"}, core.LINK_DATA_TYPE)
	db.InsertResult(two)

	timeNow += 1801
	util.SetTestTime(timeNow)
	three := core.NewSearchResult("title3", "http://url3.net",
		core.ResultPropertiesMap{core.RP_DESCRIPTION: "three"}, core.LINK_DATA_TYPE)
	db.InsertResult(three)

	timeNow += 1801
	util.SetTestTime(timeNow)
	four := core.NewSearchResult("title4", "http://url4.net",
		core.ResultPropertiesMap{core.RP_DESCRIPTION: "four"}, core.LINK_DATA_TYPE)
	db.InsertResult(four)

	five := core.NewSearchResult("title5", "http://url5.net",
		core.ResultPropertiesMap{core.RP_DESCRIPTION: "five"}, core.LINK_DATA_TYPE)
	db.InsertResult(five)

	toPublish := db.ResultsToPublish()

	if len(toPublish) < 2 {
		t.Fatalf("real size is %d", len(toPublish))
	}

	// to publish
	if toPublish[0].ResultHash != one.ResultHash {
		t.Fatal()
	}
	if toPublish[1].ResultHash != two.ResultHash {
		t.Fatal()
	}

	teardownDB()
}

// func TestDoSearch(t *testing.T) {
// 	db := setupDB()

// 	for i := 9; i >= 1; i-- {
// 		if i == 6 || i == 7 {
// 			sr := core.NewSearchResult(fmt.Sprintf("deygedygydgydgygydgygd%d", i), "",
// 				core.ResultPropertiesMap{core.RP_DESCRIPTION: fmt.Sprintf("deygedygydgydgygydgygd%d", i)},
// 				core.LINK_DATA_TYPE)
// 			db.InsertResult(sr)
// 		} else {
// 			sr := core.NewSearchResult(fmt.Sprintf("test%d", i), "",
// 				core.ResultPropertiesMap{core.RP_DESCRIPTION: fmt.Sprintf("description_test%d", i)},
// 				core.LINK_DATA_TYPE)
// 			db.InsertResult(sr)
// 		}

// 	}

// 	results := db.DoSearch("test")
// 	assertSearchResult(results[0], results[0].ResultHash, "test9", "",
// 		core.ResultPropertiesMap{core.RP_DESCRIPTION: "description_test9"}, t)
// 	assertSearchResult(results[1], results[1].ResultHash, "test8", "",
// 		core.ResultPropertiesMap{core.RP_DESCRIPTION: "description_test8"}, t)
// 	assertSearchResult(results[4], results[4].ResultHash, "test5", "",
// 		core.ResultPropertiesMap{core.RP_DESCRIPTION: "description_test5"}, t)
// 	assertSearchResult(results[5], results[5].ResultHash, "test4", "",
// 		core.ResultPropertiesMap{core.RP_DESCRIPTION: "description_test4"}, t)
// 	assertSearchResult(results[6], results[6].ResultHash, "test3", "",
// 		core.ResultPropertiesMap{core.RP_DESCRIPTION: "description_test3"}, t)
// 	assertSearchResult(results[7], results[7].ResultHash, "test2", "",
// 		core.ResultPropertiesMap{core.RP_DESCRIPTION: "description_test2"}, t)
// 	assertSearchResult(results[8], results[8].ResultHash, "test1", "",
// 		core.ResultPropertiesMap{core.RP_DESCRIPTION: "description_test1"}, t)

// 	teardownDB()
// }
