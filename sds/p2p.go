package sds

import (
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"gitlab.com/sniffdogsniff/util"
	"gitlab.com/sniffdogsniff/util/logging"
	"golang.org/x/net/proxy"

	"github.com/vmihailenco/msgpack"
)

/*
 * SniffDogSniff uses Epidemic Gossip protocol SI model
 * pull method for syncing SearchResults and Peers
 */

const (
	NONE_PROXY_TYPE        int = -1
	TOR_SOCKS_5_PROXY_TYPE int = 0
	I2P_SOCKS_5_PROXY_TYPE int = 1
)

const BUFFER_SIZE = 256
const MAX_THREAD_POOL_SIZE = 1

const FCODE_HANDSHAKE = 100
const FCODE_GET_RESULTS_FOR_SYNC = 101
const FCODE_GET_PEERS_FOR_SYNC = 102
const FCODE_GET_METADATA_FOR_SYNC = 103

const ERRCODE_NULL = 0
const ERRCODE_MARSHAL = 51
const ERRCODE_NOFUNCT = 72

func errCodeToError(funCode, errCode byte) error {
	switch errCode {
	case ERRCODE_MARSHAL:
		return errors.New("Remote error - Msgpack marshal error")
	case ERRCODE_NOFUNCT:
		return errors.New(fmt.Sprintf("Remote error - No function %d", funCode))
	}
	return nil
}

type Deque struct {
	indexes []net.Conn
}

func NewDeque() Deque {
	return Deque{indexes: make([]net.Conn, 0)}
}

func (d *Deque) push(i net.Conn) {
	d.indexes = append(d.indexes, i)
}

func (d *Deque) popFirst() net.Conn {
	conn := d.indexes[0]
	d.indexes = d.indexes[1:]
	return conn
}

func (d *Deque) isEmpty() bool {
	return 0 == len(d.indexes)
}

/*
 * receives data and decompress
 */
func receiveAndDecompress(conn net.Conn) ([]byte, int64, error) { //bytes, bytes/milliseconds, error
	recvBytes := make([]byte, 0)
	buf := make([]byte, BUFFER_SIZE)
	startTime := time.Now().UnixMilli()
	for {
		n, err := conn.Read(buf)
		if err != nil {
			break
		}
		recvBytes = append(recvBytes, buf[:n]...)
		if n < BUFFER_SIZE {
			break
		}
	}
	timeDelta := time.Now().UnixMilli() - startTime
	var speed int64
	if timeDelta != 0 {
		speed = int64(len(recvBytes)) / timeDelta
	}

	req_bytes, err := util.ZlibDecompress(recvBytes)
	if err != nil {
		return nil, speed, err
	}
	return req_bytes, speed, nil
}

/*
 * compress and sends data
 */
func compressAndSend(stream []byte, conn net.Conn) error {
	buf, err := util.ZlibCompress(stream)
	if err != nil {
		return err
	}
	_, err = conn.Write(buf)
	return err
}

type NodeServer struct {
	node      *LocalNode
	connQueue Deque
	cond      sync.Cond
}

func InitNodeServer(node *LocalNode) NodeServer {
	qLock := sync.Mutex{}
	return NodeServer{
		node:      node,
		connQueue: NewDeque(),
		cond:      *sync.NewCond(&qLock),
	}
}

func (srv *NodeServer) Serve(bindAddress string) {
	/*
	 * Initialize the request handlig function, to avoid infinite thread spawning
	 * the server works with a queued thread pool: the handler waits until one or more
	 * clients are connected then he process the request. For now we decided to leave
	 * thread pool size to 1: in future maybe we can add more threads by simply increasing
	 * MAX_THREAD_POOL_SIZE
	 */
	for tn := 0; tn < MAX_THREAD_POOL_SIZE; tn++ {
		go srv.handleAndDispatchRequests()
	}

	listener, err := net.Listen("tcp", bindAddress)
	logging.LogInfo("NodeServer is listening on", bindAddress)
	if err != nil {
		logging.LogError(err.Error())
		return
	}
	defer listener.Close()
	for true {
		conn, err := listener.Accept()
		logging.LogTrace("New connection from", conn.RemoteAddr().String())
		if err != nil {
			continue
		}
		srv.cond.L.Lock()
		srv.connQueue.push(conn)
		srv.cond.Broadcast()
		srv.cond.L.Unlock()
	}
}

func (srv *NodeServer) handleAndDispatchRequests() {
	for {
		srv.cond.L.Lock()
		for srv.connQueue.isEmpty() {
			srv.cond.Wait()
		}
		conn := srv.connQueue.popFirst()
		srv.cond.L.Unlock()

		req_bytes, _, err := receiveAndDecompress(conn)
		if err != nil {
			logging.LogError(err.Error())
			conn.Close()
			return
		}
		/*
		 * Request structure :
		 * [[function code (1byte, 0 to 255)]+[request args (msgpack marshalled, n bytes)]]
		 */

		errCode := ERRCODE_NULL
		funcCode := req_bytes[0]

		logging.LogTrace("Function request", funcCode, len(req_bytes))

		var args interface{}
		err = msgpack.Unmarshal(req_bytes[1:], &args)
		if err != nil {
			errCode = ERRCODE_MARSHAL
		}

		var returned interface{}
		switch funcCode {
		case FCODE_HANDSHAKE:
			srv.node.Handshake(args.(Peer))
		case FCODE_GET_RESULTS_FOR_SYNC:
			returned = srv.node.GetResultsForSync(args.(uint64))
		case FCODE_GET_PEERS_FOR_SYNC:
			returned = srv.node.getPeersForSync()
		case FCODE_GET_METADATA_FOR_SYNC:
			returned = srv.node.GetMetadataForSync(args.(uint64))
		default:
			returned = nil
			errCode = ERRCODE_NOFUNCT
		}

		responseBytes, err := msgpack.Marshal(returned)
		if err != nil {
			logging.LogError(err.Error())
			errCode = ERRCODE_MARSHAL
		}

		toWrite := make([]byte, 0)
		toWrite = append(toWrite, funcCode)
		toWrite = append(toWrite, byte(errCode))
		toWrite = append(toWrite, responseBytes...)

		err = compressAndSend(toWrite, conn)
		if err != nil {
			logging.LogError(err.Error())
		}
		conn.Close()
	}
}

/***************************** Peers ******************************/

type Peer struct {
	Address   string
	ProxyType int
	Rank      int64
}

func NewPeer(address string) Peer {
	return Peer{
		Address:   address,
		ProxyType: -1,
	}
}

// the LocalNode rpc method equivalent
// Note: style is Function(proxySetting ProxySetting, args) // Proxy settings are mandatory as first argument!!!
func (rn *Peer) GetResultsForSync(proxySettings ProxySettings, timestamp uint64) []SearchResult {
	searches, err := rn.callRemoteFunction(proxySettings, FCODE_GET_RESULTS_FOR_SYNC, timestamp)
	if err != nil {
		logging.LogError(err.Error())
		return nil
	}
	return searches.([]SearchResult)
}

func (rn *Peer) GetMetadataForSync(proxySettings ProxySettings, timestamp uint64) []ResultMeta {
	metadata, err := rn.callRemoteFunction(proxySettings, FCODE_GET_METADATA_FOR_SYNC, timestamp)
	if err != nil {
		logging.LogError(err.Error())
		return nil
	}
	return metadata.([]ResultMeta)
}

func (rn *Peer) GetPeersForSync(proxySettings ProxySettings) []Peer {
	peers, err := rn.callRemoteFunction(proxySettings, FCODE_GET_PEERS_FOR_SYNC, nil)
	if err != nil {
		logging.LogError(err.Error())
		return nil
	}
	return peers.([]Peer)
}

func (rn *Peer) Handshake(proxySettings ProxySettings, peer Peer) error {
	_, err := rn.callRemoteFunction(proxySettings, FCODE_HANDSHAKE, peer)
	if err != nil {
		logging.LogError(err.Error())
	}
	return err
}

func (rn *Peer) callRemoteFunction(proxySettings ProxySettings, funCode byte, args interface{}) (interface{}, error) {
	argsBytes, err := msgpack.Marshal(args)
	if err != nil {
		return nil, err
	}

	conn, err := rn.connect(proxySettings)
	if err != nil {
		logging.LogTrace("connection error")
		rn.Rank -= 500
		return nil, err
	}

	reqBytes := make([]byte, 0)
	reqBytes = append(reqBytes, funCode)
	reqBytes = append(reqBytes, argsBytes...)

	err = compressAndSend(reqBytes, conn)
	if err != nil {
		return nil, err
	}

	respBytes, speed, err := receiveAndDecompress(conn)
	if err != nil {
		conn.Close()
		rn.Rank -= 100
		return nil, err
	}
	conn.Close()

	funCode = respBytes[0]
	errCode := respBytes[1]
	speed -= int64(errCode) * 10
	rn.Rank += speed

	if errCode != ERRCODE_NULL {
		return nil, errCodeToError(funCode, errCode)
	}

	var returned interface{}
	err = msgpack.Unmarshal(respBytes, &returned)
	if err != nil {
		return nil, err
	}

	return returned, nil
}

func (rn Peer) connect(settings ProxySettings) (net.Conn, error) {
	if rn.ProxyType == NONE_PROXY_TYPE {
		return net.Dial("tcp", rn.Address)
	} else {
		dialer, err := proxy.SOCKS5("tcp", settings.AddrByType(rn.ProxyType), nil, &net.Dialer{
			Timeout:   60 * time.Second,
			KeepAlive: 30 * time.Second,
		})
		if err != nil {
			logging.LogError(err.Error())
			return nil, err
		}
		return dialer.Dial("tcp", rn.Address)
	}
}

type PeerDB struct {
	dbObject *sql.DB
}

func (sd *PeerDB) Open(path string, knownPeers []Peer) {
	sql, err := sql.Open("sqlite3", path)
	if err != nil {
		logging.LogError(err.Error())
		return
	} else {
		sd.dbObject = sql
	}
	_, err = sql.Exec("create table PEERS(ADDRESS text, PROXY_TYPE int, RANK int)")
	if err != nil {
		logging.LogWarn(err.Error())
	}

	sd.SyncFrom(knownPeers)

}

func (pdb PeerDB) GetAll() []Peer {
	return pdb.DoQuery("select * from PEERS")
}

/**
 * Gets a random peer from PeerDB (for node sync)
 */
func (pdb PeerDB) GetRandomPeer() Peer {
	peers := pdb.GetAll()
	return peers[rand.Intn(len(peers)-1)]
}

func (pdb PeerDB) SyncFrom(peers []Peer) {
	for _, p := range peers {
		pL := pdb.DoQuery(fmt.Sprintf(
			"select * from PEERS where ADDRESS='%s' and PROXY_TYPE=%d",
			p.Address, p.ProxyType))
		if len(pL) == 0 {
			pdb.insertRow(p)
		}
	}
}

func (pdb PeerDB) UpdateRank(p Peer) {
	pL := pdb.DoQuery(fmt.Sprintf(
		"select * from PEERS where ADDRESS='%s' and PROXY_TYPE=%d",
		p.Address, p.ProxyType))
	if len(pL) != 0 {
		pdb.dbObject.Exec(fmt.Sprintf("update PEERS set RANK=%d where ADDRESS='%s' and PROXY_TYPE=%d",
			p.Rank, p.Address, p.ProxyType))
	}
}

func (sd PeerDB) DoQuery(queryString string) []Peer {
	rows, err := sd.dbObject.Query(queryString)
	if err != nil {
		logging.LogError(err.Error())
		return make([]Peer, 0)
	}

	results := make([]Peer, 0)

	var address string
	var proxyType int
	var rank int64

	for rows.Next() {
		err := rows.Scan(&address, &proxyType, &rank)

		if err != nil {
			continue
		}
		results = append(results, Peer{
			Address:   address,
			ProxyType: proxyType,
			Rank:      rank,
		})
	}
	return results
}

func (pdb PeerDB) insertRow(p Peer) {
	pdb.dbObject.Exec(fmt.Sprintf(
		"insert into PEERS values('%s', %d, %d)",
		p.Address, p.ProxyType, p.Rank))
}
