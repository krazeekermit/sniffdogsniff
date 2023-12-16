package core_test

import (
	"fmt"
	"testing"

	"github.com/sniffdogsniff/core"
	"github.com/sniffdogsniff/kademlia"
	"github.com/sniffdogsniff/util"
	"github.com/vmihailenco/msgpack/v5"
)

const FAKENODE_ADDRESS = ":3000"

var RESULT1 = core.NewSearchResult("title1", "http://www.title1.com", core.ResultPropertiesMap{}, core.IMAGE_DATA_TYPE)
var RESULT2 = core.NewSearchResult("title2", "http://www.title2.com", core.ResultPropertiesMap{}, core.LINK_DATA_TYPE)

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

func (fn *fakeNode) sourceNodeConnected() bool {
	err, present := fn.args["NodeConnected"]
	if present {
		return err == nil
	}
	return present
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

func (fn *fakeNode) StoreResult(sr core.SearchResult) {
	fn.args["StoreResult"] = nil
	if sr.ResultHash != RESULT1.ResultHash || sr.Title != RESULT1.Title || sr.Url != RESULT1.Url {
		fn.args["StoreResult"] = fmt.Errorf("arguments does not match: %s != %s", sr.String(), RESULT1.String())
	}
}

func (fn *fakeNode) FindResults(query string) []core.SearchResult {
	fn.args["FindResults"] = nil
	if query != "weapon of mass destruction" {
		fn.args["FindResults"] = fmt.Errorf("arguments does not match: %s != weapon of mass destruction", query)
	}
	return []core.SearchResult{RESULT1, RESULT2}
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

func (fn *fakeNode) NodeConnected(id kademlia.KadId, addr string) {
	fn.args["NodeConnected"] = nil
	if !id.Eq(SELF_NODE.Id) {
		fn.args["NodeConnected"] = fmt.Errorf("arguments does not match: %s != %s", id, SELF_NODE.Id)
	}
	if addr != SELF_NODE.Address {
		fn.args["NodeConnected"] = fmt.Errorf("arguments does not match: %s != %s", addr, SELF_NODE.Address)
	}
}

func (fn *fakeNode) CheckNode(id kademlia.KadId, addr string) bool {
	return kademlia.NewKadIdFromAddrStr(addr).Eq(id)
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
		server.ListenTCP(FAKENODE_ADDRESS)
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

	client := core.NewNodeClient(":3000")
	_testRpc_Ping(client, t)
}

func TestRpc_Ping_1000(t *testing.T) {
	setupFakeNodeServer()

	client := core.NewNodeClient(":3000")
	for i := 0; i < 1000; i++ {
		_testRpc_Ping(client, t)
	}
}

func TestRpc_FindNode(t *testing.T) {
	setupFakeNodeServer()

	client := core.NewNodeClient(":3000")

	peers, err := client.FindNode(PEER1_ID, SELF_NODE)
	if err != nil {
		t.Fatalf(err.Error())
	}

	if !node.sourceNodeConnected() {
		t.Fatal()
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

func TestRpc_FindResults(t *testing.T) {
	setupFakeNodeServer()

	client := core.NewNodeClient(":3000")

	results, err := client.FindResults("weapon of mass destruction", SELF_NODE)
	if err != nil {
		t.Fatalf(err.Error())
	}

	if !node.sourceNodeConnected() {
		t.Fatal()
	}
	if !node.wasCalled("FindResults") {
		t.Fatal()
	}
	err = node.argsDoMatch("FindResults")
	if err != nil {
		t.Fatalf(err.Error())
	}

	assertSearchResultEq(results[0], RESULT1, t)
	assertSearchResultEq(results[1], RESULT2, t)
}

func TestRpc_GetMetadataForSync(t *testing.T) {
	setupFakeNodeServer()

	client := core.NewNodeClient(":3000")

	err := client.StoreResult(RESULT1, SELF_NODE)
	if err != nil {
		t.Fatalf(err.Error())
	}

	if !node.sourceNodeConnected() {
		t.Fatal()
	}
	if !node.wasCalled("StoreResult") {
		t.Fatal()
	}
	err = node.argsDoMatch("StoreResult")
	if err != nil {
		t.Fatalf(err.Error())
	}
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
