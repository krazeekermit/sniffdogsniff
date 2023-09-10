package core_test

import (
	"testing"

	"github.com/sniffdogsniff/core"
	"github.com/sniffdogsniff/hiddenservice"
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
	nearest map[kademlia.KadId]string
}

func (fn *fakeNode2) Ping(id kademlia.KadId, addr string) error {
	return nil
}

func (fn *fakeNode2) StoreResult(sr core.SearchResult) {
}

func (fn *fakeNode2) FindResults(query string) []core.SearchResult {
	return []core.SearchResult{}
}

func (fn *fakeNode2) NodeConnected(id kademlia.KadId, addr string) {
}

func (fn *fakeNode2) FindNode(id kademlia.KadId) map[kademlia.KadId]string {
	return fn.nearest
}

var localNode *core.LocalNode
var remoteNodes []*fakeNode2

func setupNodes() {
	if localNode != nil && remoteNodes != nil {
		return
	}

	localNode = core.NewLocalNode(core.SdsConfig{})
	localNode.KadRoutingTable().SetSelfNode(SELF_NODE)
	localNode.KadRoutingTable().PushNode(NODE_1)
	localNode.KadRoutingTable().PushNode(NODE_2)
	localNode.KadRoutingTable().PushNode(NODE_3)

	fkn1 := &fakeNode2{
		nearest: map[kademlia.KadId]string{NODE_4.Id: NODE_4.Address, NODE_9.Id: NODE_9.Address},
	}
	fkn2 := &fakeNode2{
		nearest: map[kademlia.KadId]string{NODE_5.Id: NODE_5.Address, NODE_13.Id: NODE_13.Address},
	}
	fkn3 := &fakeNode2{
		nearest: map[kademlia.KadId]string{NODE_6.Id: NODE_6.Address, NODE_1.Id: NODE_1.Address},
	}
	fkn4 := &fakeNode2{
		nearest: map[kademlia.KadId]string{NODE_7.Id: NODE_7.Address, NODE_8.Id: NODE_8.Address, NODE_9.Id: NODE_9.Address},
	}
	fkn5 := &fakeNode2{
		nearest: map[kademlia.KadId]string{NODE_10.Id: NODE_10.Address, NODE_11.Id: NODE_11.Address},
	}
	fkn6 := &fakeNode2{
		nearest: map[kademlia.KadId]string{NODE_12.Id: NODE_12.Address, NODE_13.Id: NODE_13.Address},
	}
	core.NewNodeServer(fkn1).Serve(&hiddenservice.IP4TCPProto{BindAddress: NODE_1.Address})
	core.NewNodeServer(fkn2).Serve(&hiddenservice.IP4TCPProto{BindAddress: NODE_2.Address})
	core.NewNodeServer(fkn3).Serve(&hiddenservice.IP4TCPProto{BindAddress: NODE_3.Address})
	core.NewNodeServer(fkn4).Serve(&hiddenservice.IP4TCPProto{BindAddress: NODE_4.Address})
	core.NewNodeServer(fkn5).Serve(&hiddenservice.IP4TCPProto{BindAddress: NODE_5.Address})
	core.NewNodeServer(fkn6).Serve(&hiddenservice.IP4TCPProto{BindAddress: NODE_6.Address})

	remoteNodes := make([]*fakeNode2, 6)
	remoteNodes[0] = fkn1
	remoteNodes[1] = fkn2
	remoteNodes[2] = fkn3
	remoteNodes[3] = fkn4
	remoteNodes[4] = fkn5
	remoteNodes[5] = fkn6
}

func TestNodesLookup(t *testing.T) {
	setupNodes()

	s := localNode.DoNodesLookup(SELF_NODE)
	if s != 10 {
		t.Fatalf("expect %d is %d", 10, s)
	}

	ktable := localNode.KadRoutingTable()

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
