package kademlia_test

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"github.com/sniffdogsniff/kademlia"
	"github.com/sniffdogsniff/util"
)

func TestKadId_EvalHeight(t *testing.T) {
	// 0x8737fa6d 7b6cf56b a51b26e5 00168191 2da29a1d -> exp = 159
	var k1 kademlia.KadId
	k1[0] = 0x2da29a1d
	k1[1] = 0x00168191
	k1[2] = 0xa51b26e5
	k1[3] = 0x7b6cf56b
	k1[4] = 0x8737fa6d

	k1Exp := k1.EvalHeight()

	if k1Exp != 159 {
		t.Fatalf(fmt.Sprintf("expected %d, but %d", 159, k1Exp))
	}

	k1[0] = 0x2da29a1d
	k1[1] = 0x00168191
	k1[2] = 0xa51b26e5
	k1[3] = 0x006cf56b
	k1[4] = 0x00000000

	k1Exp = k1.EvalHeight()

	if k1Exp != 118 {
		t.Fatalf(fmt.Sprintf("expected %d, but %d", 118, k1Exp))
	}

	// 0x7d152893 aba66ff1 0fc9e598 114d90d6 e013471a -> exp = 158
	k1[0] = 0xe013471a
	k1[1] = 0x114d90d6
	k1[2] = 0x0fc9e598
	k1[3] = 0xaba66ff1
	k1[4] = 0x7d152893

	k1Exp = k1.EvalHeight()

	if k1Exp != 158 {
		t.Fatalf(fmt.Sprintf("expected %d, but %d", 158, k1Exp))
	}

	// 0x1c4981b4 9e2f1fce 7c31cca6 4995a6ab e00bb366 -> exp = 156
	k1[0] = 0xe00bb366
	k1[1] = 0x4995a6ab
	k1[2] = 0x7c31cca6
	k1[3] = 0x9e2f1fce
	k1[4] = 0x1c4981b4

	k1Exp = k1.EvalHeight()

	if k1Exp != 156 {
		t.Fatalf(fmt.Sprintf("expected %d, but %d", 156, k1Exp))
	}

	// 0x066ff68a dc867899 658d13d8 20563bf4 c25f71eb -> exp = 154
	k1[0] = 0xc25f71eb
	k1[1] = 0x20563bf4
	k1[2] = 0x658d13d8
	k1[3] = 0xdc867899
	k1[4] = 0x066ff68a

	k1Exp = k1.EvalHeight()

	if k1Exp != 154 {
		t.Fatalf(fmt.Sprintf("expected %d, but %d", 154, k1Exp))
	}
	// 0x0ca57664 29660f3b 816db5ba de875db9 21e70b78 -> exp = 155
	k1[0] = 0x21e70b78
	k1[1] = 0xde875db9
	k1[2] = 0x816db5ba
	k1[3] = 0x29660f3b
	k1[4] = 0x0ca57664

	k1Exp = k1.EvalHeight()

	if k1Exp != 155 {
		t.Fatalf(fmt.Sprintf("expected %d, but %d", 155, k1Exp))
	}

	// 0x00000000 00000000 000000ba 20563bf4 21e70b78 -> exp = 71
	k1[0] = 0x21e70b78
	k1[1] = 0x20563bf4
	k1[2] = 0x000000ba
	k1[3] = 0x00000000
	k1[4] = 0x00000000

	k1Exp = k1.EvalHeight()

	if k1Exp != 71 {
		t.Fatalf(fmt.Sprintf("expected %d, but %d", 71, k1Exp))
	}
}

func TestKadId_Xor(t *testing.T) {
	var k1, k2 kademlia.KadId

	// 0x75ccc6ef 82743696 a30782d6 53303bb2 7e55fc58
	k1[0] = 0x7e55fc58
	k1[1] = 0x53303bb2
	k1[2] = 0xa30782d6
	k1[3] = 0x82743696
	k1[4] = 0x75ccc6ef

	// 0xbedeb5bb 04bbeda1 3205a691 022360b4 580e25c1
	k2[0] = 0x580e25c1
	k2[1] = 0x022360b4
	k2[2] = 0x3205a691
	k2[3] = 0x04bbeda1
	k2[4] = 0xbedeb5bb

	// k1^k2 = 0xcb127354 86cfdb37 91022447 51135b06 265bd999
	dis := k1.EvalDistance(k2)

	eq := dis[0] == 0x265bd999 && dis[1] == 0x51135b06 && dis[2] == 0x91022447 && dis[3] == 0x86cfdb37 && dis[4] == 0xcb127354
	if !eq {
		t.Fatalf("k1 ^ k2 wrong")
	}

}

func TestKadId_GenIdDistant(t *testing.T) {
	var k1 kademlia.KadId

	k1[0] = 0
	k1[1] = 0
	k1[2] = 0
	k1[3] = 0
	k1[4] = 0

	if kademlia.GenKadIdFarNBitsFrom(k1, 7)[4] != 0xfe000000 {
		t.Fatal()
	}

	if kademlia.GenKadIdFarNBitsFrom(k1, 27)[4] != 0xffffffe0 {
		t.Fatal()
	}

	if kademlia.GenKadIdFarNBitsFrom(k1, 43)[3] != 0xffe00000 {
		t.Fatal()
	}

	if kademlia.GenKadIdFarNBitsFrom(k1, 92)[2] != 0xfffffff0 {
		t.Fatal()
	}

}

func TestKBucket_AddNodes(t *testing.T) {
	kBucket := kademlia.NewKBucket(0)

	nodesList := make([]*kademlia.KNode, 25)
	for i := 0; i < 25; i++ {
		bytez := make([]byte, 20)
		rand.Read(bytez)
		randKadId := kademlia.KadIdFromBytes(bytez)
		nodesList[i] = kademlia.NewKNode(randKadId, "")
	}
	for i := 0; i < 25; i++ {
		kBucket.PushNode(nodesList[i])
	}

	nodesLen := len(kBucket.GetNodes())
	if nodesLen != 20 {
		t.Fatalf(fmt.Sprintf("nodes lenght is not %d, but is %d", 20, nodesLen))
	}

	nodesLen = len(kBucket.GetReplacementNodes())
	if nodesLen != 5 {
		t.Fatalf(fmt.Sprintf("replacements lenght is not %d, but is %d", 5, nodesLen))
	}

	for i := 0; i < 20; i++ {
		if !kBucket.GetNodes()[i].Id.Eq(nodesList[i].Id) {
			t.Fatalf(fmt.Sprintf("nodes: test node %d not in queue", i))
		}
	}
	for i := 0; i < 5; i++ {
		if !kBucket.GetReplacementNodes()[i].Id.Eq(nodesList[i+20].Id) {
			t.Fatalf(fmt.Sprintf("replacements: test node %d not in queue", i))
		}
	}

}

func TestKBucket_AddNodes_LastSeen(t *testing.T) {
	kBucket := kademlia.NewKBucket(0)

	nodesList := make([]*kademlia.KNode, 25)
	for i := 0; i < 25; i++ {
		bytez := make([]byte, 20)
		rand.Read(bytez)
		randKadId := kademlia.KadIdFromBytes(bytez)
		nodesList[i] = kademlia.NewKNode(randKadId, "")
	}
	for i := 0; i < 25; i++ {
		kBucket.PushNode(nodesList[i])
	}

	var t1 = time.Now().Unix()
	util.SetTestTime(t1)
	kBucket.PushNode(nodesList[4])
	kBucket.PushNode(nodesList[7])
	kBucket.PushNode(nodesList[2])

	util.SetTestTime(t1 + 1)
	kBucket.PushNode(nodesList[15])

	util.SetTestTime(t1 + 2)
	kBucket.PushNode(nodesList[9])

	kBucket.PushNode(nodesList[22])

	util.SetTestTime(t1 + 3)
	kBucket.PushNode(nodesList[23])

	util.SetTestTime(t1 + 4)
	kBucket.PushNode(nodesList[21])

	if kBucket.GetNodes()[4].LastSeen != uint64(t1) {
		t.Fatal()
	}
	if kBucket.GetNodes()[3].LastSeen != uint64(t1) {
		t.Fatal()
	}
	if kBucket.GetNodes()[2].LastSeen != uint64(t1) {
		t.Fatal()
	}
	if kBucket.GetNodes()[1].LastSeen != uint64(t1+1) {
		t.Fatal()
	}
	if kBucket.GetNodes()[0].LastSeen != uint64(t1+2) {
		t.Fatal()
	}
	for i := 5; i < 20; i++ {
		if kBucket.GetNodes()[i].LastSeen != 0 {
			t.Fatal()
		}
	}

	if !kBucket.GetReplacementNodes()[0].Id.Eq(nodesList[21].Id) {
		t.Fatal()
	}
	if !kBucket.GetReplacementNodes()[1].Id.Eq(nodesList[23].Id) {
		t.Fatal()
	}
	if !kBucket.GetReplacementNodes()[2].Id.Eq(nodesList[22].Id) {
		t.Fatal()
	}

}

func TestKBucket_RemoveThenAddNode_EmptyReplacementCache(t *testing.T) {
	kBucket := kademlia.NewKBucket(0)

	nodesList := make([]*kademlia.KNode, 25)
	for i := 0; i < 22; i++ {
		bytez := make([]byte, 20)
		rand.Read(bytez)
		randKadId := kademlia.KadIdFromBytes(bytez)
		nodesList[i] = kademlia.NewKNode(randKadId, "")
	}
	for i := 0; i < 20; i++ {
		kBucket.PushNode(nodesList[i])
	}

	nodesLen := len(kBucket.GetReplacementNodes())
	if nodesLen > 0 {
		t.Fatalf(fmt.Sprintf("replacements lenght is not %d, but is %d", 0, nodesLen))
	}

	for i := 0; i < 5; i++ {
		kBucket.RemoveNode(nodesList[4])
	}
	for i := 0; i < 7; i++ {
		kBucket.RemoveNode(nodesList[9])
	}
	for i := 0; i < 3; i++ {
		kBucket.RemoveNode(nodesList[2])
	}

	if kBucket.GetNodes()[4].Stales != 5 {
		t.Fatal()
	}
	if kBucket.GetNodes()[9].Stales != 7 {
		t.Fatal()
	}
	if kBucket.GetNodes()[2].Stales != 3 {
		t.Fatal()
	}

	kBucket.PushNode(nodesList[21])
	if kBucket.GetNodes()[9].Id.Eq(nodesList[9].Id) {
		t.Fatal()
	}
	if !kBucket.GetNodes()[19].Id.Eq(nodesList[21].Id) {
		t.Fatal()
	}

}

func TestKBucket_RemoveThenAddNode_WithReplacementCache(t *testing.T) {
	kBucket := kademlia.NewKBucket(0)

	nodesList := make([]*kademlia.KNode, 25)
	for i := 0; i < 25; i++ {
		bytez := make([]byte, 20)
		rand.Read(bytez)
		randKadId := kademlia.KadIdFromBytes(bytez)
		nodesList[i] = kademlia.NewKNode(randKadId, "")
	}
	for i := 0; i < 25; i++ {
		kBucket.PushNode(nodesList[i])
	}

	var t1 = time.Now().Unix()
	util.SetTestTime(t1)

	kBucket.PushNode(nodesList[22])

	util.SetTestTime(t1 + 1)
	kBucket.PushNode(nodesList[23])

	util.SetTestTime(t1 + 2)
	kBucket.PushNode(nodesList[21])

	kBucket.RemoveNode(nodesList[9])
	nodesLen := len(kBucket.GetReplacementNodes())
	if nodesLen != 4 {
		t.Fatalf(fmt.Sprintf("replacements lenght is not %d, but is %d", 4, nodesLen))
	}
	if kBucket.GetNodes()[9].Id.Eq(nodesList[9].Id) {
		t.Fatal()
	}
	if !kBucket.GetNodes()[0].Id.Eq(nodesList[21].Id) {
		t.Fatal()
	}

	if !kBucket.GetReplacementNodes()[0].Id.Eq(nodesList[23].Id) {
		t.Fatal()
	}
	if !kBucket.GetReplacementNodes()[1].Id.Eq(nodesList[22].Id) {
		t.Fatal()
	}

}

func TestKTable_PushNodes(t *testing.T) {
	// 0x8737fa6d 7b6cf56b a51b26e5 00168191 2da29a1d -> exp = 159
	var k0 kademlia.KadId
	k0[0] = 0x2da29a1d
	k0[1] = 0x00168191
	k0[2] = 0xa51b26e5
	k0[3] = 0x7b6cf56b
	k0[4] = 0x8737fa6d
	ktable := kademlia.NewKadRoutingTable()
	ktable.SetSelfNode(kademlia.NewKNode(k0, "self.onion"))

	// 0x7d152893 aba66ff1 0fc9e598 114d90d6 e013471a
	var k1 kademlia.KadId
	k1[0] = 0xe013471a
	k1[1] = 0x114d90d6
	k1[2] = 0x0fc9e598
	k1[3] = 0xaba66ff1
	k1[4] = 0x7d152893

	kn1 := kademlia.NewKNode(k1, "k1.onion")
	ktable.PushNode(kn1)

	// 0xbedeb5bb 04bbeda1 3205a691 022360b4 580e25c1
	var k2 kademlia.KadId
	k2[0] = 0x580e25c1
	k2[1] = 0x022360b4
	k2[2] = 0x3205a691
	k2[3] = 0x04bbeda1
	k2[4] = 0xbedeb5bb

	kn2 := kademlia.NewKNode(k2, "k2.onion")
	ktable.PushNode(kn2)

	// 0x966ff68a dc867899 658d13d8 20563bf4 c25f71eb
	var k3 kademlia.KadId
	k3[0] = 0xc25f71eb
	k3[1] = 0x20563bf4
	k3[2] = 0x658d13d8
	k3[3] = 0xdc867899
	k3[4] = 0x966ff68a

	kn3 := kademlia.NewKNode(k3, "k3.onion")
	ktable.PushNode(kn3)

	// 0xbf2da238 a1bd6dea 337adf2a 8e550711 f43fd7a2
	var k4 kademlia.KadId
	k4[0] = 0xf43fd7a2
	k4[1] = 0x8e550711
	k4[2] = 0x337adf2a
	k4[3] = 0xa1bd6dea
	k4[4] = 0xbf2da238

	kn4 := kademlia.NewKNode(k4, "k4.onion")
	ktable.PushNode(kn4)

	//k1 in buck 159
	if !ktable.GetKBuckets()[159].GetNodes()[0].Id.Eq(kn1.Id) {
		t.Fatal()
	}
	//k2 in buck 157
	if !ktable.GetKBuckets()[157].GetNodes()[0].Id.Eq(kn2.Id) {
		t.Fatal()
	}
	//k3 in buck 156
	if !ktable.GetKBuckets()[156].GetNodes()[0].Id.Eq(kn3.Id) {
		t.Fatal()
	}
	//k4 in buck 157
	if !ktable.GetKBuckets()[157].GetNodes()[1].Id.Eq(kn4.Id) {
		t.Fatal()
	}
}

func TestKTable_RemoveNodes(t *testing.T) {
	// 0x8737fa6d 7b6cf56b a51b26e5 00168191 2da29a1d -> exp = 159
	var k0 kademlia.KadId
	k0[0] = 0x2da29a1d
	k0[1] = 0x00168191
	k0[2] = 0xa51b26e5
	k0[3] = 0x7b6cf56b
	k0[4] = 0x8737fa6d
	ktable := kademlia.NewKadRoutingTable()
	ktable.SetSelfNode(kademlia.NewKNode(k0, "self.onion"))

	// 0x7d152893 aba66ff1 0fc9e598 114d90d6 e013471a
	var k1 kademlia.KadId
	k1[0] = 0xe013471a
	k1[1] = 0x114d90d6
	k1[2] = 0x0fc9e598
	k1[3] = 0xaba66ff1
	k1[4] = 0x7d152893

	kn1 := kademlia.NewKNode(k1, "k1.onion")
	ktable.PushNode(kn1)

	// 0xbedeb5bb 04bbeda1 3205a691 022360b4 580e25c1
	var k2 kademlia.KadId
	k2[0] = 0x580e25c1
	k2[1] = 0x022360b4
	k2[2] = 0x3205a691
	k2[3] = 0x04bbeda1
	k2[4] = 0xbedeb5bb

	kn2 := kademlia.NewKNode(k2, "k2.onion")
	ktable.PushNode(kn2)

	// 0x966ff68a dc867899 658d13d8 20563bf4 c25f71eb
	var k3 kademlia.KadId
	k3[0] = 0xc25f71eb
	k3[1] = 0x20563bf4
	k3[2] = 0x658d13d8
	k3[3] = 0xdc867899
	k3[4] = 0x966ff68a

	kn3 := kademlia.NewKNode(k3, "k3.onion")
	ktable.PushNode(kn3)

	// 0xbf2da238 a1bd6dea 337adf2a 8e550711 f43fd7a2
	var k4 kademlia.KadId
	k4[0] = 0xf43fd7a2
	k4[1] = 0x8e550711
	k4[2] = 0x337adf2a
	k4[3] = 0xa1bd6dea
	k4[4] = 0xbf2da238

	kn4 := kademlia.NewKNode(k4, "k4.onion")
	ktable.PushNode(kn4)

	if !ktable.RemoveNode(kn1) {
		t.Fatal()
	}

	if !ktable.RemoveNode(kn4) {
		t.Fatal()
	}

	//k1 in buck 159
	if ktable.GetKBuckets()[159].GetNodes()[0].Stales != 1 {
		t.Fatal()
	}
	//k4 in buck 157
	if ktable.GetKBuckets()[157].GetNodes()[1].Stales != 1 {
		t.Fatal()
	}
}

func TestKTable_GetClosest(t *testing.T) {
	// 0x8737fa6d 7b6cf56b a51b26e5 00168191 2da29a1d
	var k0 kademlia.KadId
	k0[0] = 0x2da29a1d
	k0[1] = 0x00168191
	k0[2] = 0xa51b26e5
	k0[3] = 0x7b6cf56b
	k0[4] = 0x8737fa6d
	ktable := kademlia.NewKadRoutingTable()
	kself := kademlia.NewKNode(k0, "self.onion")
	ktable.SetSelfNode(kself)

	// 0x7d152893 aba66ff1 0fc9e598 114d90d6 e013471a
	var k1 kademlia.KadId
	k1[0] = 0xe013471a
	k1[1] = 0x114d90d6
	k1[2] = 0x0fc9e598
	k1[3] = 0xaba66ff1
	k1[4] = 0x7d152893

	kn1 := kademlia.NewKNode(k1, "k1.onion")
	ktable.PushNode(kn1)

	// 0xbedeb5bb 04bbeda1 3205a691 022360b4 580e25c1
	var k2 kademlia.KadId
	k2[0] = 0x580e25c1
	k2[1] = 0x022360b4
	k2[2] = 0x3205a691
	k2[3] = 0x04bbeda1
	k2[4] = 0xbedeb5bb

	kn2 := kademlia.NewKNode(k2, "k2.onion")
	ktable.PushNode(kn2)

	// 0x966ff68a dc867899 658d13d8 20563bf4 c25f71eb
	var k3 kademlia.KadId
	k3[0] = 0xc25f71eb
	k3[1] = 0x20563bf4
	k3[2] = 0x658d13d8
	k3[3] = 0xdc867899
	k3[4] = 0x966ff68a

	kn3 := kademlia.NewKNode(k3, "k3.onion")
	ktable.PushNode(kn3)

	// 0xbf2da238 a1bd6dea 337adf2a 8e550711 f43fd7a2
	var k4 kademlia.KadId
	k4[0] = 0xf43fd7a2
	k4[1] = 0x8e550711
	k4[2] = 0x337adf2a
	k4[3] = 0xa1bd6dea
	k4[4] = 0xbf2da238

	kn4 := kademlia.NewKNode(k4, "k4.onion")
	ktable.PushNode(kn4)

	// 0x0ca57664 29660f3b 816db5ba de875db9 21e70b78 -> exp = 155
	var ka1 kademlia.KadId
	ka1[0] = 0x21e70b78
	ka1[1] = 0xde875db9
	ka1[2] = 0x816db5ba
	ka1[3] = 0x29660f3b
	ka1[4] = 0x0ca57664
	kna1 := kademlia.NewKNode(ka1, "ka1.onion")

	closest := ktable.GetNClosestTo(kna1.Id, 5)
	if !closest[0].Id.Eq(kn1.Id) {
		t.Fatal()
	}
	if !closest[1].Id.Eq(kself.Id) {
		t.Fatal()
	}
	if !closest[2].Id.Eq(kn3.Id) {
		t.Fatal()
	}
	if !closest[3].Id.Eq(kn2.Id) {
		t.Fatal()
	}
	if !closest[4].Id.Eq(kn4.Id) {
		t.Fatal()
	}
}

func TestKTable_ToBytesFromBytes(t *testing.T) {
	// 0x8737fa6d 7b6cf56b a51b26e5 00168191 2da29a1d -> exp = 159
	var k0 kademlia.KadId
	k0[0] = 0x2da29a1d
	k0[1] = 0x00168191
	k0[2] = 0xa51b26e5
	k0[3] = 0x7b6cf56b
	k0[4] = 0x8737fa6d
	ktable := kademlia.NewKadRoutingTable()
	kself := kademlia.NewKNode(k0, "self.onion")
	ktable.SetSelfNode(kself)

	// 0x7d152893 aba66ff1 0fc9e598 114d90d6 e013471a
	var k1 kademlia.KadId
	k1[0] = 0xe013471a
	k1[1] = 0x114d90d6
	k1[2] = 0x0fc9e598
	k1[3] = 0xaba66ff1
	k1[4] = 0x7d152893

	kn1 := kademlia.NewKNode(k1, "altitudegullydetoxifyjugular.onion")

	var t1 = time.Now().Unix()
	util.SetTestTime(t1)

	ktable.PushNode(kn1)
	ktable.PushNode(kn1)

	// 0xbedeb5bb 04bbeda1 3205a691 022360b4 580e25c1
	var k2 kademlia.KadId
	k2[0] = 0x580e25c1
	k2[1] = 0x022360b4
	k2[2] = 0x3205a691
	k2[3] = 0x04bbeda1
	k2[4] = 0xbedeb5bb

	kn2 := kademlia.NewKNode(k2, "crewmatesuffixjunctionbroken.onion")
	ktable.PushNode(kn2)

	// 0x966ff68a dc867899 658d13d8 20563bf4 c25f71eb
	var k3 kademlia.KadId
	k3[0] = 0xc25f71eb
	k3[1] = 0x20563bf4
	k3[2] = 0x658d13d8
	k3[3] = 0xdc867899
	k3[4] = 0x966ff68a

	kn3 := kademlia.NewKNode(k3, "tallunearthrethinkblurt.onion")
	ktable.PushNode(kn3)

	// 0xbf2da238 a1bd6dea 337adf2a 8e550711 f43fd7a2
	var k4 kademlia.KadId
	k4[0] = 0xf43fd7a2
	k4[1] = 0x8e550711
	k4[2] = 0x337adf2a
	k4[3] = 0xa1bd6dea
	k4[4] = 0xbf2da238

	kn4 := kademlia.NewKNode(k4, "thirstingcagecagesubtitle.onion")
	ktable.PushNode(kn4)

	ktable2 := kademlia.NewKadRoutingTable()

	buf := bytes.NewBuffer(nil)
	wBuf := bufio.NewWriter(buf)
	err := ktable.Write(wBuf)
	if err != nil {
		t.Fatalf(err.Error())
	}
	wBuf.Flush()
	if len(buf.Bytes()) == 0 {
		t.Fatalf("busize %d", len(buf.Bytes()))
	}
	err = ktable2.Read(bufio.NewReader(buf))
	if err != nil {
		t.Fatalf(err.Error())
	}

	// self node
	if !ktable2.SelfNode().Id.Eq(kself.Id) {
		t.Fatal()
	}
	if ktable2.SelfNode().Address != kself.Address {
		t.Fatal()
	}

	//k1 in buck 159
	if !ktable2.GetKBuckets()[159].GetNodes()[0].Id.Eq(kn1.Id) {
		t.Fatal()
	}
	if ktable2.GetKBuckets()[159].GetNodes()[0].LastSeen != uint64(t1) {
		t.Fatal()
	}
	if ktable2.GetKBuckets()[159].GetNodes()[0].Address != "altitudegullydetoxifyjugular.onion" {
		t.Fatalf(ktable2.GetKBuckets()[159].GetNodes()[0].Address)
	}
	//k2 in buck 157
	if !ktable2.GetKBuckets()[157].GetNodes()[0].Id.Eq(kn2.Id) {
		t.Fatal()
	}
	if ktable2.GetKBuckets()[157].GetNodes()[0].LastSeen != 0 {
		t.Fatal()
	}
	if ktable2.GetKBuckets()[157].GetNodes()[0].Address != "crewmatesuffixjunctionbroken.onion" {
		t.Fatal()
	}
	//k3 in buck 156
	if !ktable2.GetKBuckets()[156].GetNodes()[0].Id.Eq(kn3.Id) {
		t.Fatal()
	}
	if ktable2.GetKBuckets()[156].GetNodes()[0].Address != "tallunearthrethinkblurt.onion" {
		t.Fatal()
	}
	//k4 in buck 157
	if !ktable2.GetKBuckets()[157].GetNodes()[1].Id.Eq(kn4.Id) {
		t.Fatal()
	}
	if ktable2.GetKBuckets()[157].GetNodes()[1].Address != "thirstingcagecagesubtitle.onion" {
		t.Fatal()
	}
}
