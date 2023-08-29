package core

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/sniffdogsniff/hiddenservice"
	"github.com/sniffdogsniff/logging"
	"github.com/sniffdogsniff/proxies"
	"github.com/sniffdogsniff/util"
)

/*
 * SniffDogSniff uses Epidemic Gossip protocol SI model
 * pull method for syncing SearchResults and Peers
 */

const BUFFER_SIZE = 256
const MAX_THREAD_POOL_SIZE = 1

const (
	FCODE_HANDSHAKE             = 100
	FCODE_GETSTATUS             = 101
	FCODE_GET_RESULTS_FOR_SYNC  = 102
	FCODE_GET_PEERS_FOR_SYNC    = 103
	FCODE_GET_METADATA_FOR_SYNC = 104
	FCODE_GET_METADATA_OF       = 105
)

const (
	ERRCODE_NULL          = 000
	ERRCODE_MARSHAL       = 001
	ERRCODE_NOFUNCT       = 002
	ERRCODE_TYPE_ARGUMENT = 003
)

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

type RpcRequest struct {
	FuncCode  uint8
	Id        string // 24-bytes string
	Arguments []any
}

type RpcResponse struct {
	ErrCode  uint8
	Id       string // 24-bytes string
	RetValue any
}

type NodeServer struct {
	node      *LocalNode
	connQueue Deque
	cond      *sync.Cond
}

func NewNodeServer(node *LocalNode) *NodeServer {
	return &NodeServer{
		node:      node,
		connQueue: NewDeque(),
		cond:      sync.NewCond(&sync.Mutex{}),
	}
}

func (srv *NodeServer) Serve(proto hiddenservice.NetProtocol) {
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

	listener, err := proto.Listen()

	if err != nil {
		logging.LogError(err.Error())
		return
	}
	logging.LogInfo("NodeServer is listening on", proto.GetAddressString())

	go srv.acceptConns(listener)
}

func (srv *NodeServer) acceptConns(listener net.Listener) {
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
		var returned any

		var request RpcRequest
		if GobUnmarshal(req_bytes, &request) != nil {
			logging.LogTrace("Malformed rpc request")
			goto send_resp
		}

		logging.LogTrace("Function request", request.FuncCode, len(req_bytes))

		switch request.FuncCode {
		case FCODE_HANDSHAKE:
			peer, ok := request.Arguments[0].(Peer)
			if !ok {
				errCode = ERRCODE_TYPE_ARGUMENT
				goto send_resp
			}
			returned = srv.node.Handshake(peer)
		case FCODE_GETSTATUS:
			returned = util.TwoUint64ToArr(srv.node.GetStatus())
		case FCODE_GET_RESULTS_FOR_SYNC:
			time, ok := request.Arguments[0].(uint64)
			if !ok {
				errCode = ERRCODE_TYPE_ARGUMENT
				goto send_resp
			}
			returned = srv.node.GetResultsForSync(time)
		case FCODE_GET_PEERS_FOR_SYNC:
			returned = srv.node.getPeersForSync()
		case FCODE_GET_METADATA_FOR_SYNC:
			time, ok := request.Arguments[0].(uint64)
			if !ok {
				errCode = ERRCODE_TYPE_ARGUMENT
				goto send_resp
			}
			returned = srv.node.GetMetadataForSync(time)
		case FCODE_GET_METADATA_OF:
			hashes, ok := request.Arguments[0].([]Hash256)
			if !ok {
				errCode = ERRCODE_TYPE_ARGUMENT
				goto send_resp
			}
			returned = srv.node.GetMetadataOf(hashes)
		default:
			returned = nil
			errCode = ERRCODE_NOFUNCT
		}

	send_resp:
		response := RpcResponse{
			ErrCode:  uint8(errCode),
			Id:       request.Id,
			RetValue: returned,
		}

		responseBytes, err := GobMarshal(response)
		if err != nil {
			logging.LogTrace(err.Error())
		}

		err = compressAndSend(responseBytes, conn)
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
	ret, err := rn.callRemoteFunction(FCODE_HANDSHAKE, []any{peer})
	if err != nil {
		logging.LogError(err.Error())
	}

	remoteErr, ok := ret.(error)
	if !ok {
		logging.LogError("error: return type mismatch")
	}
	return remoteErr
}

func (rn *NodeClient) GetStatus() (uint64, uint64) {
	ret, err := rn.callRemoteFunction(FCODE_GETSTATUS, nil)
	if err != nil {
		logging.LogError(err.Error())
	}

	timestamps, ok := ret.([2]uint64)
	if !ok {
		logging.LogError("error: return type mismatch")
	}
	return util.ArrToTwoUint64(timestamps)
}

// the LocalNode rpc method equivalent
// Note: style is Function(proxySetting ProxySetting, args) // Proxy settings are mandatory as first argument!!!
func (rn *NodeClient) GetResultsForSync(timestamp uint64) []SearchResult {
	ret, err := rn.callRemoteFunction(FCODE_GET_RESULTS_FOR_SYNC, []any{timestamp})
	if err != nil {
		logging.LogError(err.Error())
		return nil
	}

	searches, ok := ret.([]SearchResult)
	if !ok {
		logging.LogError("error: return type mismatch")
	}
	return searches
}

func (rn *NodeClient) GetMetadataForSync(timestamp uint64) []ResultMeta {
	ret, err := rn.callRemoteFunction(FCODE_GET_METADATA_FOR_SYNC, []any{timestamp})
	if err != nil {
		logging.LogError(err.Error())
		return nil
	}

	metadata, ok := ret.([]ResultMeta)
	if !ok {
		logging.LogError("error: return type mismatch")
	}
	return metadata
}

func (rn *NodeClient) GetPeersForSync() []Peer {
	ret, err := rn.callRemoteFunction(FCODE_GET_PEERS_FOR_SYNC, nil)
	if err != nil {
		logging.LogError(err.Error())
		return nil
	}

	peers, ok := ret.([]Peer)
	if !ok {
		logging.LogError("error: return type mismatch")
	}
	return peers
}

func (rn *NodeClient) GetMetadataOf(hashes []Hash256) []ResultMeta {
	var metadata []ResultMeta
	ret, err := rn.callRemoteFunction(FCODE_GET_METADATA_OF, []any{hashes})
	if err != nil {
		logging.LogError(err.Error())
		return nil
	}

	metadata, ok := ret.([]ResultMeta)
	if !ok {
		logging.LogError("error: return type mismatch")
	}
	return metadata
}

func (rn *NodeClient) callRemoteFunction(funCode byte, args []any) (any, error) {
	conn, err := rn.proxySettings.NewConnection(rn.peer.Address)
	if err != nil {
		logging.LogTrace("connection error")
		return nil, err
	}

	request := RpcRequest{
		FuncCode:  funCode,
		Id:        util.GenerateId12_Str(),
		Arguments: args,
	}

	reqBytes, err := GobMarshal(request)
	if err != nil {
		return nil, err
	}

	err = compressAndSend(reqBytes, conn)
	if err != nil {
		return nil, err
	}

	respBytes, _, err := receiveAndDecompress(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}
	conn.Close()

	var response RpcResponse
	err = GobUnmarshal(respBytes, &response)
	if err != nil {
		return nil, err
	}

	if response.Id != request.Id {
		return nil, errors.New("rpc error: the request and response id did not match")
	}

	if response.ErrCode != ERRCODE_NULL {
		return nil, errCodeToError(request.FuncCode, response.ErrCode)
	}

	return response.RetValue, nil
}
