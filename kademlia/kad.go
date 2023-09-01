package kademlia

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"fmt"
	"sort"
	"time"
)

const K int = 20
const KAD_ID_LEN int = 160
const STALES_THR uint = 5 //to be changed

type KadId [5]uint32 // is a big integer of 160-bit lenght (5 x 32-bit unsigned int)

func NewKadId(addr string) KadId {
	bytez := make([]byte, 20)
	for i, ib := range sha1.Sum([]byte(addr)) {
		bytez[i] = ib
	}

	return KadIdFromBytes(bytez)
}

func KadIdFromBytes(bz []byte) KadId {
	var a KadId
	a[0] = uint32(bz[0]) | (uint32(bz[1]) << 8) | (uint32(bz[2]) << 16) | (uint32(bz[3]) << 24)
	a[1] = uint32(bz[4]) | (uint32(bz[5]) << 8) | (uint32(bz[6]) << 16) | (uint32(bz[7]) << 24)
	a[2] = uint32(bz[8]) | (uint32(bz[9]) << 8) | (uint32(bz[10]) << 16) | (uint32(bz[11]) << 24)
	a[3] = uint32(bz[12]) | (uint32(bz[13]) << 8) | (uint32(bz[14]) << 16) | (uint32(bz[15]) << 24)
	a[4] = uint32(bz[16]) | (uint32(bz[17]) << 8) | (uint32(bz[18]) << 16) | (uint32(bz[19]) << 24)
	return a
}

func (a KadId) ToBytes() []byte {
	return []byte{
		byte(a[0]), byte(a[0] >> 8), byte(a[0] >> 16), byte(a[0] >> 24),
		byte(a[1]), byte(a[1] >> 8), byte(a[1] >> 16), byte(a[1] >> 24),
		byte(a[2]), byte(a[2] >> 8), byte(a[2] >> 16), byte(a[2] >> 24),
		byte(a[3]), byte(a[3] >> 8), byte(a[3] >> 16), byte(a[3] >> 24),
		byte(a[4]), byte(a[4] >> 8), byte(a[4] >> 16), byte(a[4] >> 24),
	}
}

func (a KadId) Eq(b KadId) bool {
	return a[0] == b[0] && a[1] == b[1] && a[2] == b[2] && a[3] == b[3] && a[4] == b[4]
}

func (a KadId) EvalDistance(b KadId) KadId {
	var d KadId
	d[0] = a[0] ^ b[0]
	d[1] = a[1] ^ b[1]
	d[2] = a[2] ^ b[2]
	d[3] = a[3] ^ b[3]
	d[4] = a[4] ^ b[4]
	return d
}

func (a KadId) lessThan(b KadId) bool {
	if a[4] != b[4] {
		return a[4] < b[4]
	}
	if a[3] != b[3] {
		return a[3] < b[3]
	}
	if a[2] != b[2] {
		return a[2] < b[2]
	}
	if a[1] != b[1] {
		return a[1] < b[1]
	}
	return a[0] < b[0]
}

// evaluates the bucket height for the id -> 2^height < id < 2^height+1
func (a KadId) EvalHeight() int {
	for s := 0; s < KAD_ID_LEN; s++ {
		i := KAD_ID_LEN - 1 - s
		num := a[i/32]
		if num == 0 {
			s += 31
			continue
		}

		if (num & (0x7fffffff >> (s % 32))) != num {
			return i
		}
	}
	return 0
}

type KNode struct {
	Id       KadId
	LastSeen uint64
	Stales   uint
	Address  string
}

func NewKNode(id KadId, addr string) *KNode {
	return &KNode{
		Id:       id,
		LastSeen: 0,
		Stales:   0,
		Address:  addr,
	}
}

func (kn *KNode) SeenNow() {
	kn.LastSeen = uint64(time.Now().Unix())
}

/*
	KBucket
*/

func hasNode(list []*KNode, kn *KNode) (int, bool) {
	if len(list) == 0 {
		return -1, false
	}
	for i, ikn := range list {
		if kn.Id.Eq(ikn.Id) {
			return i, true
		}
	}
	return -1, false
}

func remove(list []*KNode, idx int) []*KNode {
	lastIdx := len(list) - 1
	if idx < 0 || idx > lastIdx {
		return list
	}

	if idx == 0 { //for efficiency
		return list[1:]
	}
	if idx == (len(list) - 1) { //for efficiency
		return list[:lastIdx]
	}

	return append(list[:idx], list[idx+1:]...)
}

type KBucket struct {
	nodes            []*KNode
	replacementNodes []*KNode
	height           uint
}

func NewKBucket(height uint) *KBucket {
	return &KBucket{
		nodes:            make([]*KNode, 0),
		replacementNodes: make([]*KNode, 0),
		height:           height,
	}
}

func (kbuck *KBucket) sortReplacements() {
	sort.Slice(kbuck.replacementNodes, func(i, j int) bool {
		return kbuck.replacementNodes[i].LastSeen > kbuck.replacementNodes[j].LastSeen
	})
}

// skips the push logic only used internally
func (kbuck *KBucket) insertNode(kn *KNode) {
	kbuck.nodes = append(kbuck.nodes, kn)
}

// skips the push logic only used internally
func (kbuck *KBucket) insertReplacementNode(kn *KNode) {
	kbuck.replacementNodes = append(kbuck.replacementNodes, kn)
}

func (kbuck *KBucket) GetNodes() []*KNode {
	return kbuck.nodes
}

func (kbuck *KBucket) GetReplacementNodes() []*KNode {
	return kbuck.replacementNodes
}

func (kbuck *KBucket) PushNode(kn *KNode) {
	idx, present := hasNode(kbuck.nodes, kn)
	if present {
		kbuck.nodes[idx].SeenNow()
		kbuck.nodes[idx].Stales = 0
	} else if len(kbuck.nodes) < K {
		kbuck.nodes = append(kbuck.nodes, kn)
	} else {
		stalestIdx := -1
		var staleMax uint = 0
		for i, kn := range kbuck.nodes {
			if kn.Stales >= STALES_THR && kn.Stales > staleMax {
				stalestIdx = i
				staleMax = kn.Stales
			}
		}

		if stalestIdx > -1 {
			kbuck.nodes = remove(kbuck.nodes, stalestIdx)
			kbuck.nodes = append(kbuck.nodes, kn)
		} else {
			ridx, present := hasNode(kbuck.replacementNodes, kn)
			if present {
				kbuck.replacementNodes[ridx].SeenNow()
			} else if len(kbuck.replacementNodes) < K {
				kbuck.replacementNodes = append(kbuck.replacementNodes, kn)
			} else {
				kbuck.replacementNodes = remove(kbuck.replacementNodes, len(kbuck.replacementNodes)-1)
				kbuck.replacementNodes = append(kbuck.replacementNodes, kn)
			}
			kbuck.sortReplacements()
		}
	}
}

func (kbuck *KBucket) RemoveNode(kn *KNode) bool {
	idx, present := hasNode(kbuck.nodes, kn)
	if present {
		if len(kbuck.replacementNodes) > 0 {
			kbuck.nodes = remove(kbuck.nodes, idx)
			first := kbuck.replacementNodes[0]
			kbuck.nodes = append(kbuck.nodes, first)
			kbuck.replacementNodes = remove(kbuck.replacementNodes, 0)
			kbuck.sortReplacements()
		} else {
			kbuck.nodes[idx].Stales++
		}
	}
	return present
}

/*
 Ktable (former peerDB)
*/

type KadRoutingTable struct {
	self     *KNode
	kbuckets []*KBucket
}

func NewKadRoutingTable(self *KNode) *KadRoutingTable {
	ktable := KadRoutingTable{
		self:     self,
		kbuckets: make([]*KBucket, KAD_ID_LEN),
	}

	for i := 0; i < KAD_ID_LEN; i++ {
		ktable.kbuckets[i] = NewKBucket(uint(i))
	}
	return &ktable
}

func (ktable *KadRoutingTable) GetKBuckets() []*KBucket {
	return ktable.kbuckets
}

func (ktable *KadRoutingTable) PushNode(kn *KNode) {
	ktable.kbuckets[kn.Id.EvalDistance(ktable.self.Id).EvalHeight()].PushNode(kn)
}

func (ktable *KadRoutingTable) RemoveNode(kn *KNode) bool {
	return ktable.kbuckets[kn.Id.EvalDistance(ktable.self.Id).EvalHeight()].RemoveNode(kn)
}

func (ktable *KadRoutingTable) GetNClosestOf(kn *KNode, n int) []*KNode {
	allNodes := make([]*KNode, 0)
	for _, bucket := range ktable.kbuckets {
		allNodes = append(allNodes, bucket.nodes...)
	}
	sort.Slice(allNodes, func(i, j int) bool {
		iDistance := allNodes[i].Id.EvalDistance(kn.Id)
		jDistance := allNodes[j].Id.EvalDistance(kn.Id)
		return iDistance.lessThan(jDistance)
	})
	return allNodes[:n+1]
}

func (ktable *KadRoutingTable) GetNClosest(n int) []*KNode {
	return ktable.GetNClosestOf(ktable.self, n)
}

func (ktable *KadRoutingTable) ToBytes() []byte {
	//ktableHeader [height(1byte)][nentries(1byte)]
	//knode_row [replacement(1byte)][kad_id(20 bytes)][lastseen(8bytes)][stales(1byte)][address(n-bytes)]

	buffer := bytes.NewBuffer(nil)
	for _, bucket := range ktable.kbuckets {
		buffer.WriteByte(byte(bucket.height))
		buffer.WriteByte(byte(len(bucket.nodes) + len(bucket.replacementNodes)))
		for _, node := range bucket.nodes {
			buffer.WriteByte(0)
			buffer.Write(node.Id.ToBytes())
			buffer.Write(binary.LittleEndian.AppendUint64([]byte{}, node.LastSeen))
			buffer.WriteByte(byte(node.Stales))
			buffer.WriteByte(byte(len(node.Address)))
			buffer.Write([]byte(node.Address))
		}
		for _, node := range bucket.replacementNodes {
			buffer.WriteByte(1)
			buffer.Write(node.Id.ToBytes())
			buffer.Write(binary.LittleEndian.AppendUint64([]byte{}, node.LastSeen))
			buffer.WriteByte(byte(node.Stales))
			buffer.WriteByte(byte(len(node.Address)))
			buffer.Write([]byte(node.Address))
		}
	}

	return buffer.Bytes()
}

func (ktable *KadRoutingTable) FromBytes(bytez []byte) error {
	//ktableHeader [height(1byte)][nentries(1byte)]
	//knode_row [replacement(1byte)][kad_id(20 bytes)][lastseen(8bytes)][stales(1byte)][address(n-bytes)]

	buffer := bytes.NewBuffer(bytez)
	for i := 0; i < 160; i++ {
		height, err := buffer.ReadByte()
		if err != nil {
			return err
		}
		size, err := buffer.ReadByte()
		if err != nil {
			return err
		}
		for j := 0; j < int(size); j++ {
			replacementByte, err := buffer.ReadByte()
			if err != nil {
				return err
			}
			kadIdBytez := make([]byte, 20)
			n, err := buffer.Read(kadIdBytez)
			if err != nil {
				return err
			}
			if n != 20 {
				return errors.New(" read: n != 20")
			}
			lastSeenBytez := make([]byte, 8)
			n, err = buffer.Read(lastSeenBytez)
			if err != nil {
				return err
			}
			if n != 8 {
				return errors.New(" read: n != 8")
			}
			stalesByte, err := buffer.ReadByte()
			if err != nil {
				return err
			}
			addressStrSize, err := buffer.ReadByte()
			if err != nil {
				return err
			}
			addressBytez := make([]byte, addressStrSize)
			n, err = buffer.Read(addressBytez)
			if err != nil {
				return err
			}
			if n != int(addressStrSize) {
				return fmt.Errorf(" read: n != %d", addressStrSize)
			}

			kn := KNode{
				Id:       KadIdFromBytes(kadIdBytez),
				LastSeen: binary.LittleEndian.Uint64(lastSeenBytez),
				Stales:   uint(stalesByte),
				Address:  string(addressBytez),
			}

			if replacementByte == 1 {
				ktable.kbuckets[height].insertReplacementNode(&kn)
			} else {
				ktable.kbuckets[height].insertNode(&kn)
			}
		}
		ktable.kbuckets[height].sortReplacements()
	}

	return nil
}
