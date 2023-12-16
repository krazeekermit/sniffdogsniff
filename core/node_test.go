package core_test

import (
	"strings"
	"testing"

	"github.com/sniffdogsniff/core"
	"github.com/sniffdogsniff/kademlia"
)

var SELF_NODE = kademlia.NewKNode(kademlia.NewKadId("node00"), "node00")
var NODE_1 = kademlia.NewKNode(kademlia.NewKadId(":3001"), ":3001")    // 156-bucket
var NODE_2 = kademlia.NewKNode(kademlia.NewKadId(":3002"), ":3002")    // 156-bucket
var NODE_3 = kademlia.NewKNode(kademlia.NewKadId(":3003"), ":3003")    // 158-bucket
var NODE_4 = kademlia.NewKNode(kademlia.NewKadId(":3004"), ":3004")    // 158-bucket
var NODE_5 = kademlia.NewKNode(kademlia.NewKadId(":3005"), ":3005")    // 159-bucket
var NODE_6 = kademlia.NewKNode(kademlia.NewKadId(":3006"), ":3006")    // 156-bucket
var NODE_7 = kademlia.NewKNode(kademlia.NewKadId("node7"), "node7")    // 158-bucket
var NODE_8 = kademlia.NewKNode(kademlia.NewKadId("node8"), "node8")    // 157-bucket
var NODE_9 = kademlia.NewKNode(kademlia.NewKadId("node9"), "node9")    // 158-bucket
var NODE_10 = kademlia.NewKNode(kademlia.NewKadId("node10"), "node10") // 158-bucket
var NODE_11 = kademlia.NewKNode(kademlia.NewKadId("node11"), "node11") // 159-bucket
var NODE_12 = kademlia.NewKNode(kademlia.NewKadId("node12"), "node12") // 156-bucket
var NODE_13 = kademlia.NewKNode(kademlia.NewKadId("node13"), "node13") // 157-bucket

var NODES_MAP = map[int][]*kademlia.KNode{
	156: {NODE_1, NODE_2, NODE_6, NODE_12},
	157: {NODE_8, NODE_13},
	158: {NODE_3, NODE_4, NODE_7, NODE_9, NODE_10},
	159: {NODE_5, NODE_11},
}

type fakeNode2 struct {
	name    string
	nearest map[kademlia.KadId]string
	store   []core.SearchResult
}

func (fn *fakeNode2) Ping(id kademlia.KadId, addr string) error {
	return nil
}

func (fn *fakeNode2) StoreResult(sr core.SearchResult) {
	for _, isr := range fn.store {
		if sr.ResultHash == isr.ResultHash {
			return
		}
	}
	fn.store = append(fn.store, sr)
}

func (fn *fakeNode2) FindResults(query string) []core.SearchResult {
	found := make([]core.SearchResult, 0)
	for _, isr := range fn.store {
		if strings.Contains(isr.Title, query) {
			found = append(found, isr)
		}
	}
	return found
}

func (fn *fakeNode2) NodeConnected(id kademlia.KadId, addr string) {
}

func (fn *fakeNode2) CheckNode(id kademlia.KadId, addr string) bool {
	check := kademlia.NewKadIdFromAddrStr(addr).Eq(id)
	return check
}

func (fn *fakeNode2) FindNode(id kademlia.KadId) map[kademlia.KadId]string {
	return fn.nearest
}

var localNode *core.LocalNode
var remoteNodes []*fakeNode2

func setupNodes() {
	if localNode != nil && len(remoteNodes) == 6 {
		return
	}

	setupTestDir()
	localNode = core.NewLocalNode(core.SdsConfig{WorkDirPath: test_dir})
	clearTestDir()
	localNode.KadRoutingTable().SetSelfNode(SELF_NODE)
	localNode.KadRoutingTable().PushNode(NODE_1)
	localNode.KadRoutingTable().PushNode(NODE_2)
	localNode.KadRoutingTable().PushNode(NODE_3)

	fkn1 := &fakeNode2{
		name:    "Node1",
		store:   make([]core.SearchResult, 0),
		nearest: map[kademlia.KadId]string{NODE_4.Id: NODE_4.Address, NODE_9.Id: NODE_9.Address},
	}
	fkn2 := &fakeNode2{
		name:    "Node2",
		store:   make([]core.SearchResult, 0),
		nearest: map[kademlia.KadId]string{NODE_5.Id: NODE_5.Address, NODE_13.Id: NODE_13.Address},
	}
	fkn3 := &fakeNode2{
		name:    "Node3",
		store:   make([]core.SearchResult, 0),
		nearest: map[kademlia.KadId]string{NODE_6.Id: NODE_6.Address, NODE_1.Id: NODE_1.Address},
	}
	fkn4 := &fakeNode2{
		name:    "Node4",
		store:   make([]core.SearchResult, 0),
		nearest: map[kademlia.KadId]string{NODE_7.Id: NODE_7.Address, NODE_8.Id: NODE_8.Address, NODE_9.Id: NODE_9.Address},
	}
	fkn5 := &fakeNode2{
		name:    "Node5",
		store:   make([]core.SearchResult, 0),
		nearest: map[kademlia.KadId]string{NODE_10.Id: NODE_10.Address, NODE_11.Id: NODE_11.Address},
	}
	fkn6 := &fakeNode2{
		name:    "Node6",
		store:   make([]core.SearchResult, 0),
		nearest: map[kademlia.KadId]string{NODE_12.Id: NODE_12.Address, NODE_13.Id: NODE_13.Address},
	}

	remoteNodes = make([]*fakeNode2, 6)
	remoteNodes[0] = fkn1
	remoteNodes[1] = fkn2
	remoteNodes[2] = fkn3
	remoteNodes[3] = fkn4
	remoteNodes[4] = fkn5
	remoteNodes[5] = fkn6

	core.NewNodeServer(fkn1).ListenTCP(NODE_1.Address)
	core.NewNodeServer(fkn2).ListenTCP(NODE_2.Address)
	core.NewNodeServer(fkn3).ListenTCP(NODE_3.Address)
	core.NewNodeServer(fkn4).ListenTCP(NODE_4.Address)
	core.NewNodeServer(fkn5).ListenTCP(NODE_5.Address)
	core.NewNodeServer(fkn6).ListenTCP(NODE_6.Address)
}

func TestNodesLookup(t *testing.T) {
	setupNodes()

	localNode.DoNodesLookup(SELF_NODE, false)

	ktable := localNode.KadRoutingTable()

	// bucketNodes := ktable.GetKBuckets()
	// for i, kb := range bucketNodes {
	// 	for _, n := range kb.GetNodes() {
	// 		fmt.Printf("-> bucket %d, node %s\n", i, n.Address)
	// 	}
	// }

	for ib, nodes := range NODES_MAP {
		bucketNodes := ktable.GetKBuckets()[ib].GetNodes()
		if len(bucketNodes) != len(nodes) {
			t.Fatalf("bucket %d, size %d != expected %d", ib, len(bucketNodes), len(nodes))
		}

		for in, kn := range bucketNodes {
			if !bucketNodes[in].Id.Eq(kn.Id) {
				t.Fatalf("bucket %d, node %d", ib, in)
			}
		}
	}

	for ib, nodes := range NODES_MAP {
		bucketNodes := ktable.GetKBuckets()[ib].GetNodes()
		if len(bucketNodes) != len(nodes) {
			t.Fatalf("bucket %d, size %d != expected %d", ib, len(bucketNodes), len(nodes))
		}
		for in, kn := range bucketNodes {
			if !bucketNodes[in].Id.Eq(kn.Id) {
				t.Fatalf("bucket %d, node %d", ib, in)
			}
			stales := 1
			if kn.Id.Eq(NODE_1.Id) || kn.Id.Eq(NODE_2.Id) || kn.Id.Eq(NODE_3.Id) || kn.Id.Eq(NODE_4.Id) || kn.Id.Eq(NODE_5.Id) || kn.Id.Eq(NODE_6.Id) {
				stales = 0
			}
			if bucketNodes[in].Stales != uint(stales) {
				t.Fatalf("stales != %d::%d, node %d", stales, ib, in)
			}
		}
	}

}

func TestDoSearch(t *testing.T) {
	setupNodes()
	sr := core.NewSearchResult("A weapon of mass destruction (WMD) is a chemical, biological, radiological or nuclear.",
		"http:://title1.ycom", core.ResultPropertiesMap{}, core.LINK_DATA_TYPE)

	remoteNodes[0].store = make([]core.SearchResult, 0)
	remoteNodes[1].store = append(remoteNodes[1].store, sr)
	remoteNodes[2].store = append(remoteNodes[2].store, sr)

	if len(remoteNodes[0].store) != 0 {
		t.Fatal()
	}

	results := localNode.DoSearch("mass destruction")

	if len(results) != 1 {
		t.Fatal()
	}
	if results[0].ResultHash != sr.ResultHash {
		t.Fatal()
	}

	if len(remoteNodes[0].store) == 0 {
		t.Fatal()
	}
	if remoteNodes[0].store[0].ResultHash != sr.ResultHash {
		t.Fatal()
	}
}

func TestPublishResult(t *testing.T) {
	setupNodes()
	sr := core.NewSearchResult("title1 1 2 3", "http:://title1.com", core.ResultPropertiesMap{}, core.LINK_DATA_TYPE)
	sr.QueryMetrics[0] = NODE_1.Id
	sr.QueryMetrics[1] = NODE_2.Id
	sr.QueryMetrics[2] = NODE_3.Id

	remoteNodes[0].store = make([]core.SearchResult, 0)
	remoteNodes[1].store = make([]core.SearchResult, 0)
	remoteNodes[2].store = make([]core.SearchResult, 0)

	localNode.PublishResults([]core.SearchResult{sr})

	if len(remoteNodes[0].store) == 0 {
		t.Fatal()
	}
	if remoteNodes[0].store[0].ResultHash != sr.ResultHash {
		t.Fatal()
	}

	if len(remoteNodes[1].store) == 0 {
		t.Fatal()
	}
	if remoteNodes[1].store[0].ResultHash != sr.ResultHash {
		t.Fatal()
	}

	if len(remoteNodes[2].store) == 0 {
		t.Fatal()
	}
	if remoteNodes[2].store[0].ResultHash != sr.ResultHash {
		t.Fatal()
	}
}
