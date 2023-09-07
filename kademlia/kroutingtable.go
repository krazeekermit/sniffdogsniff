package kademlia

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/sniffdogsniff/util"
)

const KTABLE_FILENAME = "ktable.dat"

/*
 Ktable (former peerDB)
*/

func SortNodesByDistance(id KadId, nodes []*KNode) {
	sort.Slice(nodes, func(i, j int) bool {
		iDistance := nodes[i].Id.EvalDistance(id)
		jDistance := nodes[j].Id.EvalDistance(id)
		return iDistance.LessThan(jDistance)
	})
}

type KadRoutingTable struct {
	self     *KNode
	filePath string
	kbuckets []*KBucket
}

func NewKadRoutingTable(self *KNode) *KadRoutingTable {
	ktable := KadRoutingTable{
		self:     self,
		filePath: "",
		kbuckets: make([]*KBucket, KAD_ID_LEN),
	}

	for i := 0; i < KAD_ID_LEN; i++ {
		ktable.kbuckets[i] = NewKBucket(uint(i))
	}
	return &ktable
}

func (ktable *KadRoutingTable) SetSelfNode(self *KNode) {
	ktable.self = self
}

func (ktable *KadRoutingTable) GetKBuckets() []*KBucket {
	return ktable.kbuckets
}

func (ktable *KadRoutingTable) IsEmpty() bool {
	for i := 0; i < KAD_ID_LEN; i++ {
		if !ktable.kbuckets[i].isEmpty() {
			return false
		}
	}
	return true
}

func (ktable *KadRoutingTable) IsFull() bool {
	for i := 0; i < KAD_ID_LEN; i++ {
		if !ktable.kbuckets[i].isFull() {
			return false
		}
	}
	return true
}

func (ktable *KadRoutingTable) PushNode(kn *KNode) {
	ktable.kbuckets[kn.Id.EvalDistance(ktable.self.Id).EvalHeight()].PushNode(kn)
}

func (ktable *KadRoutingTable) RemoveNode(kn *KNode) bool {
	return ktable.kbuckets[kn.Id.EvalDistance(ktable.self.Id).EvalHeight()].RemoveNode(kn)
}

func (ktable *KadRoutingTable) GetNClosestTo(targetId KadId, n int) []*KNode {
	allNodes := make([]*KNode, 0)
	for _, bucket := range ktable.kbuckets {
		allNodes = append(allNodes, bucket.nodes...)
	}
	SortNodesByDistance(targetId, allNodes)

	if len(allNodes) < n {
		n = len(allNodes)
	}
	return allNodes[:n]
}

func (ktable *KadRoutingTable) GetNClosest(n int) []*KNode {
	return ktable.GetNClosestTo(ktable.self.Id, n)
}

func (ktable *KadRoutingTable) GetKClosest() []*KNode {
	return ktable.GetNClosestTo(ktable.self.Id, K)
}

func (ktable *KadRoutingTable) GetKClosestTo(id KadId) []*KNode {
	return ktable.GetNClosestTo(id, K)
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

func (ktable *KadRoutingTable) Open(workDirPath string) error {
	ktable.filePath = filepath.Join(workDirPath, KTABLE_FILENAME)
	if util.FileExists(ktable.filePath) {
		fp, err := os.OpenFile(ktable.filePath, os.O_RDONLY, 0600)
		if err != nil {
			return err
		}
		bytez := make([]byte, 0)
		for {
			buf := make([]byte, 2048)
			n, err := fp.Read(buf)
			if err != nil {
				fp.Close()
				return err
			}
			bytez = append(bytez, buf[:n]...)
			if n < 2048 {
				break
			}
		}
		ktable.FromBytes(bytez)
		fp.Close()
	}
	return nil
}

func (ktable *KadRoutingTable) Flush() error {
	fp, err := os.OpenFile(ktable.filePath, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	fp.Write(ktable.ToBytes())
	fp.Close()
	return nil
}
