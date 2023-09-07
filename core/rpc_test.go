package core_test

import (
	"fmt"
	"testing"

	"github.com/sniffdogsniff/core"
	"github.com/sniffdogsniff/hiddenservice"
	"github.com/sniffdogsniff/kademlia"
	"github.com/sniffdogsniff/proxies"
	"github.com/sniffdogsniff/util"
	"github.com/vmihailenco/msgpack/v5"
)

const FAKENODE_ADDRESS = ":3000"

var RESULT1 = core.NewSearchResult("title1", "http://www.title1.com", core.ResultPropertiesMap{}, core.IMAGE_DATA_TYPE)
var RESULT2 = core.NewSearchResult("title2", "http://www.title2.com", core.ResultPropertiesMap{}, core.LINK_DATA_TYPE)

var RMETA1 = core.NewResultMeta(RESULT1.ResultHash, 1234, 23, 0)
var RMETA2 = core.NewResultMeta(RESULT2.ResultHash, 7654, 89, 1)

var PEER1_ADDR = "thirstingcagecagesubtitle.onion"
var PEER2_ADDR = "tallunearthrethinkblurt.onion"

var PEER1_ID = kademlia.NewKadId(PEER1_ADDR)
var PEER2_ID = kademlia.NewKadId(PEER2_ADDR)

type fakeNode struct {
	args map[string]error
}

func (fn *fakeNode) wasCalled(funcName string) bool {
	_, present := fn.args[funcName]
	return present
}

func (fn *fakeNode) argsDoMatch(funcName string) error {
	return fn.args[funcName]
}

func (fn *fakeNode) Ping(id kademlia.KadId, addr string) error {
	fn.args["Ping"] = nil
	if !id.Eq(PEER1_ID) {
		fn.args["Ping"] = fmt.Errorf("arguments does not match: %s != %s", id, PEER1_ID)
	}
	if addr != PEER1_ADDR {
		fn.args["Ping"] = fmt.Errorf("arguments does not match: %s != %s", addr, PEER1_ADDR)
	}
	return nil
}

func (fn *fakeNode) GetStatus() (uint64, uint64) {
	fn.args["GetStatus"] = nil
	return 1936, 1441
}

func (fn *fakeNode) GetResultsForSync(timestamp uint64) []core.SearchResult {
	fn.args["GetResultsForSync"] = nil
	if timestamp != 1936 {
		fn.args["GetResultsForSync"] = fmt.Errorf("arguments does not match: %d != %d", timestamp, 1936)
	}
	return []core.SearchResult{RESULT1, RESULT2}
}

func (fn *fakeNode) GetMetadataForSync(ts uint64) []core.ResultMeta {
	fn.args["GetMetadataForSync"] = nil
	if ts != 4567 {
		fn.args["GetMetadataForSync"] = fmt.Errorf("arguments does not match: %d != %d", ts, 4567)
	}
	return []core.ResultMeta{RMETA1, RMETA2}
}

func (fn *fakeNode) GetMetadataOf(hashes []core.Hash256) []core.ResultMeta {
	fn.args["GetMetadataOf"] = nil
	if len(hashes) != 2 {
		fn.args["GetMetadataOf"] = fmt.Errorf("sizeof hashes %d != %d", len(hashes), 2)
		return []core.ResultMeta{}
	}
	if hashes[0] != RESULT1.ResultHash {
		fn.args["GetMetadataOf"] = fmt.Errorf("arguments does not match: %s != %s", hashes[0], RESULT1.ResultHash)
	}
	if hashes[1] != RESULT2.ResultHash {
		fn.args["GetMetadataOf"] = fmt.Errorf("arguments does not match: %s != %s", hashes[1], RESULT2.ResultHash)
	}
	return []core.ResultMeta{RMETA1, RMETA2}
}

func (fn *fakeNode) FindNode(id kademlia.KadId) map[kademlia.KadId]string {
	fn.args["FindNode"] = nil
	if !id.Eq(PEER1_ID) {
		fn.args["FindNode"] = fmt.Errorf("arguments does not match: %s != %s", id, PEER1_ID)
	}
	return map[kademlia.KadId]string{
		PEER1_ID: PEER1_ADDR,
		PEER2_ID: PEER2_ADDR,
	}
}

var server *core.NodeServer = nil
var node *fakeNode = nil

func setupFakeNodeServer() {
	if node == nil {
		node = &fakeNode{
			make(map[string]error),
		}
	}

	if server == nil {
		server = core.NewNodeServer(node)
		server.Serve(&hiddenservice.IP4TCPProto{BindAddress: FAKENODE_ADDRESS})
	}

}

func _testRpc_Ping(client core.NodeClient, t *testing.T) {
	_, err := client.Ping(PEER1_ID, PEER1_ADDR)
	if err != nil {
		t.Fatalf(err.Error())
	}

	if !node.wasCalled("Ping") {
		t.Fatal()
	}
	err = node.argsDoMatch("Ping")
	if err != nil {
		t.Fatalf(err.Error())
	}
}

func TestRpc_Ping(t *testing.T) {
	setupFakeNodeServer()

	client := core.NewNodeClient(":3000", proxies.ProxySettings{})
	_testRpc_Ping(client, t)
}

func TestRpc_Ping_1000(t *testing.T) {
	setupFakeNodeServer()

	client := core.NewNodeClient(":3000", proxies.ProxySettings{})
	for i := 0; i < 1000; i++ {
		_testRpc_Ping(client, t)
	}
}

func _testRpc_GetStatus(client core.NodeClient, t *testing.T) {
	s1, s2, err := client.GetStatus()
	if err != nil {
		t.Fatalf(err.Error())
	}

	if !node.wasCalled("GetStatus") {
		t.Fatal()
	}
	err = node.argsDoMatch("GetStatus")
	if err != nil {
		t.Fatalf(err.Error())
	}

	if s1 != 1936 {
		t.Fatal()
	}
	if s2 != 1441 {
		t.Fatal()
	}
}

func TestRpc_GetStatus(t *testing.T) {
	setupFakeNodeServer()

	client := core.NewNodeClient(":3000", proxies.ProxySettings{})
	_testRpc_GetStatus(client, t)
}

func TestRpc_GetStatus_1000(t *testing.T) {
	setupFakeNodeServer()

	client := core.NewNodeClient(":3000", proxies.ProxySettings{})
	for i := 0; i < 1000; i++ {
		_testRpc_GetStatus(client, t)
	}
}

func TestRpc_FindNode(t *testing.T) {
	setupFakeNodeServer()

	client := core.NewNodeClient(":3000", proxies.ProxySettings{})

	peers, err := client.FindNode(PEER1_ID)
	if err != nil {
		t.Fatalf(err.Error())
	}

	if !node.wasCalled("FindNode") {
		t.Fatal()
	}
	err = node.argsDoMatch("FindNode")
	if err != nil {
		t.Fatalf(err.Error())
	}

	if peers[PEER1_ID] != PEER1_ADDR {
		t.Fatal()
	}
	if peers[PEER2_ID] != PEER2_ADDR {
		t.Fatal()
	}
}

func TestRpc_GetResultsForSync(t *testing.T) {
	setupFakeNodeServer()

	client := core.NewNodeClient(":3000", proxies.ProxySettings{})

	results, err := client.GetResultsForSync(1936)
	if err != nil {
		t.Fatalf(err.Error())
	}

	if !node.wasCalled("GetResultsForSync") {
		t.Fatal()
	}
	err = node.argsDoMatch("GetResultsForSync")
	if err != nil {
		t.Fatalf(err.Error())
	}

	assertSearchResultEq(results[0], RESULT1, t)
	assertSearchResultEq(results[1], RESULT2, t)
}

func TestRpc_GetMetadataForSync(t *testing.T) {
	setupFakeNodeServer()

	client := core.NewNodeClient(":3000", proxies.ProxySettings{})

	metas, err := client.GetMetadataForSync(4567)
	if err != nil {
		t.Fatalf(err.Error())
	}

	if !node.wasCalled("GetMetadataForSync") {
		t.Fatal()
	}
	err = node.argsDoMatch("GetMetadataForSync")
	if err != nil {
		t.Fatalf(err.Error())
	}

	assertMetaEq(metas[0], RMETA1, t)
	assertMetaEq(metas[1], RMETA2, t)
}

func TestRpc_GetMetadataOf(t *testing.T) {
	setupFakeNodeServer()

	client := core.NewNodeClient(":3000", proxies.ProxySettings{})

	metas, err := client.GetMetadataOf([]core.Hash256{RESULT1.ResultHash, RESULT2.ResultHash})
	if err != nil {
		t.Fatalf(err.Error())
	}

	if !node.wasCalled("GetMetadataOf") {
		t.Fatal()
	}
	err = node.argsDoMatch("GetMetadataOf")
	if err != nil {
		t.Fatalf(err.Error())
	}

	assertMetaEq(metas[0], RMETA1, t)
	assertMetaEq(metas[1], RMETA2, t)
}

func TestMarshalUnmarshal_RpcRequest(t *testing.T) {
	orig := core.RpcRequest{
		FuncCode: 134,
		Id:       util.GenerateId12_Str(),
	}

	bytez, err := msgpack.Marshal(orig)
	if err != nil {
		t.Fatal()
	}

	if len(bytez) != 40 {
		t.Fatal()
	}

	var req core.RpcRequest
	err = msgpack.Unmarshal(bytez, &req)
	if err != nil {
		t.Fatal()
	}

	if req.FuncCode != orig.FuncCode {
		t.Fatal()
	}
	if req.Id != orig.Id {
		t.Fatal()
	}
}

func TestMarshalUnmarshal_RpcResponse(t *testing.T) {
	orig := core.RpcResponse{
		ErrCode: 67,
		Id:      util.GenerateId12_Str(),
	}

	bytez, err := msgpack.Marshal(orig)
	if err != nil {
		t.Fatal()
	}

	if len(bytez) != 39 {
		t.Fatal()
	}

	var req core.RpcResponse
	err = msgpack.Unmarshal(bytez, &req)
	if err != nil {
		t.Fatal()
	}

	if req.ErrCode != orig.ErrCode {
		t.Fatal()
	}
	if req.Id != orig.Id {
		t.Fatal()
	}
}
