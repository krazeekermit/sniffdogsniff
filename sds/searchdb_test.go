package sds_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/sniffdogsniff/sds"
)

const TEST_FILE = "./test.db"

func setupDB() sds.SearchDB {
	//logging.InitLogging(logging.TRACE)
	db := sds.SearchDB{}
	db.Open(TEST_FILE, 1024*1024*256)
	return db
}

func teardownDB() {
	err := os.Remove(TEST_FILE)
	if err != nil {
		fmt.Println(err.Error())
	}
}

func differentValues(name string, a, b interface{}, t *testing.T) {
	t.Fatal("Different", name, "values: wanted", b, "actual", a)
}

func assertMetaRecord(meta sds.ResultMeta, rHash [32]byte, score int, inv int, t *testing.T) {
	if meta.ResultHash != rHash {
		differentValues("hash", meta.ResultHash, rHash, t)
	}
	if meta.Score != score {
		differentValues("score", meta.Score, score, t)
	}
	// if meta.Invalidated != inv {
	// 	t.Fatalf("invalidated is different!")
	// }
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
	metas1 = append(metas1, sds.NewResultMetadata(one.ResultHash, 12, sds.INVALIDATED))
	metas1 = append(metas1, sds.NewResultMetadata(two.ResultHash, 23, sds.INVALIDATED))

	db.SyncResultsMetadataFrom(metas1)
	db.Flush()

	all := db.GetDB().GetAll()
	if len(all) > 1 {
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

	dbLen := len(db.GetCacheDB().GetAll())
	if !(dbLen == 5) {
		t.Fatalf("lenght of db != 5, actual: %d", dbLen)
	}

	teardownDB()
}

func TestTimeBasedSync_Meta(t *testing.T) {
	db := setupDB()
	one := sds.NewSearchResult("title1", "http://url1.net", "one")
	db.InsertResult(one)
	two := sds.NewSearchResult("title2", "http://url2.net", "two")
	db.InsertResult(two)
	db.InsertResult(sds.NewSearchResult("title3", "http://url3.net", "three"))

	metas1 := make([]sds.ResultMeta, 0)
	metas1 = append(metas1, sds.NewResultMetadata(one.ResultHash, 12, 25))
	metas1 = append(metas1, sds.NewResultMetadata(two.ResultHash, 23, 0))

	db.SyncResultsMetadataFrom(metas1)

	time.Sleep(2 * time.Second)

	db.SyncResultsMetadataFrom([]sds.ResultMeta{sds.NewResultMetadata(one.ResultHash, 11, 21)})

	metasDB := db.GetCacheDB().GetAllMetadata()
	assertMetaRecord(metasDB[one.ResultHash], one.ResultHash, (11+12)/2, 21, t)

	teardownDB()
}
