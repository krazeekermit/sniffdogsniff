package sds

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/sniffdogsniff/proxies"
	"github.com/sniffdogsniff/util"
	"github.com/sniffdogsniff/util/logging"

	"github.com/vmihailenco/msgpack"
)

/*
 * SniffDogSniff uses Epidemic Gossip protocol SI model
 * pull method for syncing SearchResults and Peers
 */

const BUFFER_SIZE = 256
const MAX_THREAD_POOL_SIZE = 1

const FCODE_HANDSHAKE = 100
const FCODE_GETSTATUS = 101
const FCODE_GET_RESULTS_FOR_SYNC = 102
const FCODE_GET_PEERS_FOR_SYNC = 103
const FCODE_GET_METADATA_FOR_SYNC = 104
const FCODE_GET_METADATA_OF = 105

const ERRCODE_NULL = 0
const ERRCODE_MARSHAL = 51
const ERRCODE_NOFUNCT = 72

func errCodeToError(funCode, errCode byte) error {
	switch errCode {
	case ERRCODE_MARSHAL:
		return errors.New("remote error - msgpack marshal error")
	case ERRCODE_NOFUNCT:
		return fmt.Errorf("remote error - no function %d", funCode)
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
	return len(d.indexes) == 0
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
	cond      *sync.Cond
}

func InitNodeServer(node *LocalNode) NodeServer {
	return NodeServer{
		node:      node,
		connQueue: NewDeque(),
		cond:      sync.NewCond(&sync.Mutex{}),
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
	for {
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

		argsBytes := req_bytes[1:]

		var returned interface{}
		switch funcCode {
		case FCODE_HANDSHAKE:
			var arg Peer
			err := msgpack.Unmarshal(argsBytes, &arg)
			if err != nil {
				errCode = ERRCODE_MARSHAL
			} else {
				returned = srv.node.Handshake(arg)
			}
		case FCODE_GETSTATUS:
			returned = util.TwoUint64ToArr(srv.node.GetStatus())
		case FCODE_GET_RESULTS_FOR_SYNC:
			var arg uint64
			err := msgpack.Unmarshal(argsBytes, &arg)
			if err != nil {
				errCode = ERRCODE_MARSHAL
			} else {
				returned = srv.node.GetResultsForSync(arg)
			}
		case FCODE_GET_PEERS_FOR_SYNC:
			returned = srv.node.getPeersForSync()
		case FCODE_GET_METADATA_FOR_SYNC:
			var arg uint64
			err := msgpack.Unmarshal(argsBytes, &arg)
			if err != nil {
				errCode = ERRCODE_MARSHAL
			} else {
				returned = srv.node.GetMetadataForSync(arg)
			}
		case FCODE_GET_METADATA_OF:
			var arg []Hash256
			err := msgpack.Unmarshal(argsBytes, &arg)
			if err != nil {
				errCode = ERRCODE_MARSHAL
			} else {
				returned = srv.node.GetMetadataOf(arg)
			}
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

type NodeClient struct {
	peer          Peer
	proxySettings proxies.ProxySettings
}

func NewNodeClient(peer Peer, proxySettings proxies.ProxySettings) NodeClient {
	return NodeClient{
		peer:          peer,
		proxySettings: proxySettings,
	}
}

/***************************** Remote Node (Client) ******************************/

func (rn *NodeClient) Handshake(peer Peer) error {
	var remoteErr interface{}
	err := rn.callRemoteFunction(FCODE_HANDSHAKE, peer, &remoteErr)
	if err != nil {
		logging.LogError(err.Error())
	}
	return err
}

func (rn *NodeClient) GetStatus() (uint64, uint64) {
	var timestamps [2]uint64
	err := rn.callRemoteFunction(FCODE_GETSTATUS, nil, &timestamps)
	if err != nil {
		logging.LogError(err.Error())
	}
	return util.ArrToTwoUint64(timestamps)
}

// the LocalNode rpc method equivalent
// Note: style is Function(proxySetting ProxySetting, args) // Proxy settings are mandatory as first argument!!!
func (rn *NodeClient) GetResultsForSync(timestamp uint64) []SearchResult {
	var searches []SearchResult
	err := rn.callRemoteFunction(FCODE_GET_RESULTS_FOR_SYNC, timestamp, &searches)
	if err != nil {
		logging.LogError(err.Error())
		return nil
	}
	return searches
}

func (rn *NodeClient) GetMetadataForSync(timestamp uint64) []ResultMeta {
	var metadata []ResultMeta
	err := rn.callRemoteFunction(FCODE_GET_METADATA_FOR_SYNC, timestamp, &metadata)
	if err != nil {
		logging.LogError(err.Error())
		return nil
	}
	return metadata
}

func (rn *NodeClient) GetPeersForSync() []Peer {
	var peers []Peer
	err := rn.callRemoteFunction(FCODE_GET_PEERS_FOR_SYNC, nil, &peers)
	if err != nil {
		logging.LogError(err.Error())
		return nil
	}
	return peers
}

func (rn *NodeClient) GetMetadataOf(hashes []Hash256) []ResultMeta {
	var metadata []ResultMeta
	err := rn.callRemoteFunction(FCODE_GET_METADATA_OF, hashes, &metadata)
	if err != nil {
		logging.LogError(err.Error())
		return nil
	}
	return metadata
}

func (rn *NodeClient) callRemoteFunction(funCode byte, args interface{}, returned interface{}) error {
	argsBytes, err := msgpack.Marshal(args)
	if err != nil {
		return err
	}

	conn, err := rn.proxySettings.NewConnection(rn.peer.Address)
	if err != nil {
		logging.LogTrace("connection error")
		rn.peer.Rank -= 500
		return err
	}

	reqBytes := make([]byte, 0)
	reqBytes = append(reqBytes, funCode)
	reqBytes = append(reqBytes, argsBytes...)

	err = compressAndSend(reqBytes, conn)
	if err != nil {
		return err
	}

	respBytes, speed, err := receiveAndDecompress(conn)
	if err != nil {
		conn.Close()
		rn.peer.Rank -= 100
		return err
	}
	conn.Close()

	funCode = respBytes[0]
	errCode := respBytes[1]
	speed -= int64(errCode) * 10
	rn.peer.Rank += speed

	if errCode != ERRCODE_NULL {
		return errCodeToError(funCode, errCode)
	}

	err = msgpack.Unmarshal(respBytes[2:], returned)
	if err != nil {
		return err
	}

	return nil
}
