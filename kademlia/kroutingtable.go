package kademlia

import (
	"bufio"
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
	selfNode *KNode
	filePath string
	kbuckets []*KBucket
}

func NewKadRoutingTable() *KadRoutingTable {
	ktable := KadRoutingTable{
		selfNode: nil,
		filePath: "",
		kbuckets: make([]*KBucket, KAD_ID_LEN),
	}
	ktable.init()
	return &ktable
}

func (ktable *KadRoutingTable) init() {
	for i := 0; i < KAD_ID_LEN; i++ {
		ktable.kbuckets[i] = NewKBucket(uint(i))
	}
}

func (ktable *KadRoutingTable) allNodes() []*KNode {
	allNodes := make([]*KNode, 0)
	for _, bucket := range ktable.kbuckets {
		allNodes = append(allNodes, bucket.nodes...)
	}
	return allNodes
}

func (ktable *KadRoutingTable) SelfNode() *KNode {
	return ktable.selfNode
}

func (ktable *KadRoutingTable) SetSelfNode(self *KNode) {
	if ktable.selfNode == nil || !ktable.selfNode.Id.Eq(self.Id) {
		ktable.selfNode = self
		allNodes := ktable.allNodes()
		ktable.init()
		for _, ikn := range allNodes {
			ktable.PushNode(ikn)
		}
	}
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
	if kn.Id.Eq(ktable.selfNode.Id) {
		return
	}
	ktable.kbuckets[kn.Id.EvalDistance(ktable.selfNode.Id).EvalHeight()].PushNode(kn)
}

func (ktable *KadRoutingTable) RemoveNode(kn *KNode) bool {
	return ktable.kbuckets[kn.Id.EvalDistance(ktable.selfNode.Id).EvalHeight()].RemoveNode(kn)
}

func (ktable *KadRoutingTable) GetNClosestTo(targetId KadId, n int) []*KNode {
	allNodes := ktable.allNodes()

	// avoids adding self node with distance 0
	if !targetId.Eq(ktable.selfNode.Id) {
		allNodes = append(allNodes, ktable.selfNode)
	}

	SortNodesByDistance(targetId, allNodes)

	if len(allNodes) < n {
		n = len(allNodes)
	}
	return allNodes[:n]
}

func (ktable *KadRoutingTable) GetNClosest(n int) []*KNode {
	return ktable.GetNClosestTo(ktable.selfNode.Id, n)
}

func (ktable *KadRoutingTable) GetKClosest() []*KNode {
	return ktable.GetNClosestTo(ktable.selfNode.Id, K)
}

func (ktable *KadRoutingTable) GetKClosestTo(id KadId) []*KNode {
	return ktable.GetNClosestTo(id, K)
}

func (ktable *KadRoutingTable) Write(buffer *bufio.Writer) error {
	//selfNode [kad_id(20bytes)][address(n-bytes)]
	//ktableHeader [height(1byte)][nentries(1byte)]
	//knode_row [replacement(1byte)][kad_id(20 bytes)][lastseen(8bytes)][stales(1byte)][address(n-bytes)]

	//self node
	_, err := buffer.Write(ktable.selfNode.Id.ToBytes())
	if err != nil {
		return err
	}
	err = buffer.WriteByte(byte(len(ktable.selfNode.Address)))
	if err != nil {
		return err
	}
	_, err = buffer.Write([]byte(ktable.selfNode.Address))
	if err != nil {
		return err
	}
	//kbuckets
	for _, bucket := range ktable.kbuckets {
		err = buffer.WriteByte(byte(bucket.height))
		if err != nil {
			return err
		}
		err = buffer.WriteByte(byte(len(bucket.nodes) + len(bucket.replacementNodes)))
		if err != nil {
			return err
		}
		for _, node := range bucket.nodes {
			err = buffer.WriteByte(0)
			if err != nil {
				return err
			}
			_, err = buffer.Write(node.Id.ToBytes())
			if err != nil {
				return err
			}
			_, err = buffer.Write(binary.LittleEndian.AppendUint64([]byte{}, node.LastSeen))
			if err != nil {
				return err
			}
			err = buffer.WriteByte(byte(node.Stales))
			if err != nil {
				return err
			}
			err = buffer.WriteByte(byte(len(node.Address)))
			if err != nil {
				return err
			}
			_, err = buffer.Write([]byte(node.Address))
			if err != nil {
				return err
			}
		}
		for _, node := range bucket.replacementNodes {
			err = buffer.WriteByte(1)
			if err != nil {
				return err
			}
			_, err = buffer.Write(node.Id.ToBytes())
			if err != nil {
				return err
			}
			_, err = buffer.Write(binary.LittleEndian.AppendUint64([]byte{}, node.LastSeen))
			if err != nil {
				return err
			}
			err = buffer.WriteByte(byte(node.Stales))
			if err != nil {
				return err
			}
			err = buffer.WriteByte(byte(len(node.Address)))
			if err != nil {
				return err
			}
			_, err = buffer.Write([]byte(node.Address))
			if err != nil {
				return err
			}
		}

	}

	return nil
}

func (ktable *KadRoutingTable) Read(buffer *bufio.Reader) error {
	//ktableHeader [height(1byte)][nentries(1byte)]
	//knode_row [replacement(1byte)][kad_id(20 bytes)][lastseen(8bytes)][stales(1byte)][address(n-bytes)]

	//self node
	kadIdBytez := make([]byte, 20)
	n, err := buffer.Read(kadIdBytez)
	if err != nil {
		return err
	}
	if n != 20 {
		return errors.New(" read: n != 20")
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
	ktable.selfNode = NewKNode(KadIdFromBytes(kadIdBytez), string(addressBytez))
	//kbuckets
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
		ktable.kbuckets[height].sort()
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

		defer fp.Close()
		return ktable.Read(bufio.NewReader(fp))

	}
	return nil
}

func (ktable *KadRoutingTable) Flush() error {
	fp, err := os.OpenFile(ktable.filePath, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer fp.Close()
	return ktable.Write(bufio.NewWriter(fp))
}
