package sds

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"math/rand"
	"os"
	"path/filepath"

	"github.com/sniffdogsniff/util"
	"github.com/sniffdogsniff/util/logging"
)

const PEER_DB_DIR_NAME = "peers"
const PEER_NAME_TEMPLATE = "peer_%d.dat"

func hash(bytez []byte) uint32 {
	h := fnv.New32a()
	h.Write(bytez)
	return h.Sum32()
}

type Peer struct {
	Address   string
	ProxyType int
	Rank      int64
}

func NewPeer(address string) Peer {
	return Peer{
		Address:   address,
		ProxyType: -1,
		Rank:      -1,
	}
}

func BytesToPeer(bytez []byte) (Peer, error) {
	buf := bytes.NewBuffer(bytez)
	addrLen, err := buf.ReadByte() // ADDRESS
	if err != nil {
		return Peer{}, err
	}
	baddr := make([]byte, addrLen)
	n, err := buf.Read(baddr)
	if err != nil || n != int(addrLen) {
		return Peer{}, err
	}
	proxyType, err := buf.ReadByte()
	if err != nil {
		return Peer{}, err
	}
	brank := make([]byte, 8) // RANK
	n, err = buf.Read(brank)
	if err != nil || n != 8 {
		return Peer{}, err
	}

	return Peer{
		Address:   string(baddr),
		ProxyType: int(int8(proxyType)),
		Rank:      int64(binary.LittleEndian.Uint64(brank)),
	}, nil
}

func (p *Peer) ToBytes() []byte {
	buf := bytes.NewBuffer(nil)
	buf.WriteByte(byte(len(p.Address)))
	buf.Write([]byte(p.Address))
	buf.WriteByte(byte(p.ProxyType))
	rankBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(rankBytes, uint64(p.Rank))
	buf.Write(rankBytes)
	return buf.Bytes()
}

type PeerDB struct {
	dbPath string
	peers  map[string]Peer
}

func (pdb *PeerDB) Open(workDir string, knownPeers []Peer) {
	pdb.peers = make(map[string]Peer)

	pdb.dbPath = filepath.Join(workDir, PEER_DB_DIR_NAME)

	if util.DirExists(pdb.dbPath) {
		peersFiles, err := os.ReadDir(pdb.dbPath)
		if err != nil {
			logging.LogError("No DB found [peers]")
			panic(err)
		}
		for _, peerFile := range peersFiles {
			path := filepath.Join(pdb.dbPath, peerFile.Name())
			fp, err := os.Open(path)

			psize, err := fp.Seek(0, os.SEEK_END)
			fp.Seek(0, os.SEEK_SET)
			if err != nil {
				logging.LogError("Peer file Corrupted:", path, err)
			}
			prbytes := make([]byte, psize)
			fp.Read(prbytes)
			p, err := BytesToPeer(prbytes)
			if err != nil {
				logging.LogError("Peer file Corrupted:", path, err)
				fp.Close()
			}
			fp.Close()

			pdb.peers[p.Address] = p
		}
	} else {
		pdb.SyncFrom(knownPeers)
		logging.LogWarn("Peer database files does not exists...")
	}
}

func (pdb *PeerDB) GetAll() []Peer {
	return util.MapToSlice(pdb.peers)
}

/**
 * Gets a random peer from PeerDB (for node sync)
 */
func (pdb *PeerDB) GetRandomPeer() Peer {
	return pdb.GetAll()[rand.Intn(len(pdb.peers))]
}

func (pdb *PeerDB) SyncFrom(peers []Peer) {
	for _, p := range peers {
		pdb.peers[p.Address] = p
	}
}

func (pdb *PeerDB) Flush() {
	if !util.DirExists(pdb.dbPath) {
		os.Mkdir(pdb.dbPath, 0700)
	}

	for _, p := range pdb.peers {
		pbytez := p.ToBytes()
		fname := fmt.Sprintf(PEER_NAME_TEMPLATE, hash(pbytez))
		fp, err := os.OpenFile(filepath.Join(pdb.dbPath, fname), os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			logging.LogError("err writing peer:", filepath.Join(pdb.dbPath, fname), err)
			fp.Close()
			continue
		}
		fp.Write(pbytez)
		fp.Close()
	}
}
