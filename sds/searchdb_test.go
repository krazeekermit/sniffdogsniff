package sds_test

import (
	"bytes"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/sniffdogsniff/sds"
	"github.com/sniffdogsniff/util/logging"
	"github.com/vmihailenco/msgpack"
)

const TEST_DIR = "./test_dir"

func setupDB() sds.SearchDB {
	logging.InitLogging(logging.TRACE)
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

func assertMetaRecord(meta sds.ResultMeta, rHash [32]byte, score int, inv int, t *testing.T) {
	if meta.ResultHash != rHash {
		differentValues("hash", meta.ResultHash, rHash, t)
	}
	if meta.Score != uint16(score) {
		differentValues("score", meta.Score, score, t)
	}
	// if meta.Invalidated != inv {
	// 	t.Fatalf("invalidated is different!")
	// }
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

// func TestDeleteInvalidated(t *testing.T) {
// 	db := setupDB()
// 	one := sds.NewSearchResult("title1", "http://url1.net", "one")
// 	db.InsertResult(one)
// 	two := sds.NewSearchResult("title2", "http://url2.net", "two")
// 	db.InsertResult(two)
// 	three := sds.NewSearchResult("title3", "http://url3.net", "three")
// 	db.InsertResult(three)

// 	metas1 := make([]sds.ResultMeta, 0)
// 	metas1 = append(metas1, sds.NewResultMetadata(one.ResultHash, 12, sds.INVALIDATED))
// 	metas1 = append(metas1, sds.NewResultMetadata(two.ResultHash, 23, sds.INVALIDATED))

// 	db.SyncResultsMetadataFrom(metas1)
// 	db.Flush()

// 	all := db.GetDB().GetAll()
// 	if len(all) > 1 {
// 		t.Fatalf("DB size wrong: %d", len(all))
// 	}
// 	if all[three.ResultHash] != three {
// 		t.Fatalf("want three eq three")
// 	}

// 	teardownDB()
// }

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

// func TestTimeBasedSync_Meta(t *testing.T) {
// 	db := setupDB()
// 	one := sds.NewSearchResult("title1", "http://url1.net", "one")
// 	db.InsertResult(one)
// 	two := sds.NewSearchResult("title2", "http://url2.net", "two")
// 	db.InsertResult(two)
// 	db.InsertResult(sds.NewSearchResult("title3", "http://url3.net", "three"))

// 	metas1 := make([]sds.ResultMeta, 0)
// 	metas1 = append(metas1, sds.NewResultMetadata(one.ResultHash, 12, 25))
// 	metas1 = append(metas1, sds.NewResultMetadata(two.ResultHash, 23, 0))

// 	db.SyncResultsMetadataFrom(metas1)

// 	time.Sleep(2 * time.Second)

// 	db.SyncResultsMetadataFrom([]sds.ResultMeta{sds.NewResultMetadata(one.ResultHash, 11, 21)})

// 	metasDB := db.GetCacheDB().GetAllMetadata()
// 	assertMetaRecord(metasDB[one.ResultHash], one.ResultHash, (11+12)/2, 21, t)

// 	teardownDB()
// }

func TestStressInsertion_Searches(t *testing.T) {
	db := setupDB()

	toSync := make([]sds.SearchResult, 0)

	init_time := time.Now().Unix()
	for i := 0; i < 1048576; /*512 MB*/ i++ {
		toSync = append(toSync, sds.NewSearchResult(
			fmt.Sprintf("Title%d", i), fmt.Sprintf("http://url%d.net", i),
			fmt.Sprintf("DescriptionsBlaBlaBlaBlaBla%d", i),
		))
	}
	fmt.Println(time.Now().Unix() - init_time)

	db.SyncFrom(toSync)

	// dbLen := len(db.GetSearchesCache())
	// if !(dbLen == 5) {
	// 	t.Fatalf("lenght of db != 5, actual: %d", dbLen)
	// }

	teardownDB()
}

func TestMarshal_Unmarshal_msgpack(t *testing.T) {

	peer1 := sds.NewPeer("www.idontknow.com")

	var buffer bytes.Buffer
	enc := msgpack.NewEncoder(&buffer)
	enc.Encode(&peer1)

	rcvBuf := bytes.NewBuffer(buffer.Bytes())
	dec := msgpack.NewDecoder(rcvBuf)

	v, _ := dec.DecodeInterface()

	fmt.Println(v)

}
