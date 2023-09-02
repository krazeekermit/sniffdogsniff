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
	"github.com/vmihailenco/msgpack/v5"
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
	case ERRCODE_TYPE_ARGUMENT:
		return fmt.Errorf("remote error - unmarshal args fail for funct %d", funCode)
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

const RPC_REQ_BYTESIZE int = 40

type RpcRequest struct {
	FuncCode uint8
	Id       string // 24-bytes string
}

const RPC_RESP_BYTESIZE int = 39

type RpcResponse struct {
	ErrCode uint8
	Id      string // 24-bytes string
}

type NodeServer struct {
	node      NodeInterface
	connQueue Deque
	cond      *sync.Cond
}

func NewNodeServer(node NodeInterface) *NodeServer {
	return &NodeServer{
		node:      node,
		connQueue: NewDeque(),
		cond:      sync.NewCond(&sync.Mutex{}),
	}
}

type PingArgs struct {
	Pinger Peer
}

type PingReply struct {
	Error error
}

func (srv *NodeServer) ping(args PingArgs, reply *PingReply) {
	(*reply).Error = srv.node.Ping(args.Pinger)
}

type GetPeersForSyncArgs struct {
}

type GetPeersForSyncReply struct {
	Peers []Peer
}

func (srv *NodeServer) getPeersForSync(args GetPeersForSyncArgs, reply *GetPeersForSyncReply) {
	(*reply).Peers = srv.node.GetPeersForSync()
}

type GetStatusArgs struct {
}

type GetStatusReply struct {
	LastTimestamp     uint64
	LastMetaTimestamp uint64
}

func (srv *NodeServer) getStatus(args GetStatusArgs, reply *GetStatusReply) {
	lts, lmts := srv.node.GetStatus()
	(*reply).LastTimestamp = lts
	(*reply).LastMetaTimestamp = lmts
}

type TimestampArgs struct {
	Timestamp uint64
}

type GetResultsForSyncReply struct {
	Results []SearchResult
}

func (srv *NodeServer) getResultsForSync(args TimestampArgs, reply *GetResultsForSyncReply) {
	(*reply).Results = srv.node.GetResultsForSync(args.Timestamp)
}

type GetMetadataForSyncReply struct {
	Metadatas []ResultMeta
}

func (srv *NodeServer) getMetadataForSync(args TimestampArgs, reply *GetMetadataForSyncReply) {
	(*reply).Metadatas = srv.node.GetMetadataForSync(args.Timestamp)
}

type GetMetadataOfArgs struct {
	Hashes []Hash256
}

func (srv *NodeServer) getMetadataOf(args GetMetadataOfArgs, reply *GetMetadataForSyncReply) {
	(*reply).Metadatas = srv.node.GetMetadataOf(args.Hashes)
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

		recvBytez, _, err := receiveAndDecompress(conn)
		if err != nil {
			logging.LogError(err.Error())
			conn.Close()
			return
		}
		/*
		 * Request structure : [RpcRequest(40); Args ....]
		 */

		var request RpcRequest
		var argsBytes []byte
		var returned interface{}

		errCode := ERRCODE_NULL

		if len(recvBytez) <= RPC_REQ_BYTESIZE {
			errCode = ERRCODE_MARSHAL
			goto send_resp
		}

		if msgpack.Unmarshal(recvBytez[:RPC_REQ_BYTESIZE], &request) != nil {
			logging.LogTrace("Malformed rpc request")
			errCode = ERRCODE_MARSHAL
			goto send_resp
		}
		argsBytes = recvBytez[RPC_REQ_BYTESIZE:]

		logging.LogTrace("Function request", request.FuncCode, len(recvBytez))

		switch request.FuncCode {
		case FCODE_HANDSHAKE:
			var args PingArgs
			err := msgpack.Unmarshal(argsBytes, &args)
			if err != nil {
				logging.LogTrace(err.Error())
				errCode = ERRCODE_TYPE_ARGUMENT
				goto send_resp
			}
			var reply PingReply
			srv.ping(args, &reply)
			returned = reply
		case FCODE_GETSTATUS:
			var args GetStatusArgs
			err := msgpack.Unmarshal(argsBytes, &args)
			if err != nil {
				errCode = ERRCODE_TYPE_ARGUMENT
				goto send_resp
			}
			var reply GetStatusReply
			srv.getStatus(args, &reply)
			returned = reply
		case FCODE_GET_RESULTS_FOR_SYNC:
			var args TimestampArgs
			err := msgpack.Unmarshal(argsBytes, &args)
			if err != nil {
				errCode = ERRCODE_TYPE_ARGUMENT
				goto send_resp
			}
			var reply GetResultsForSyncReply
			srv.getResultsForSync(args, &reply)
			returned = reply
		case FCODE_GET_PEERS_FOR_SYNC:
			var args GetPeersForSyncArgs
			err := msgpack.Unmarshal(argsBytes, &args)
			if err != nil {
				logging.LogTrace(err.Error())
				errCode = ERRCODE_TYPE_ARGUMENT
				goto send_resp
			}
			var reply GetPeersForSyncReply
			srv.getPeersForSync(args, &reply)
			returned = reply
		case FCODE_GET_METADATA_FOR_SYNC:
			var args TimestampArgs
			err := msgpack.Unmarshal(argsBytes, &args)
			if err != nil {
				errCode = ERRCODE_TYPE_ARGUMENT
				goto send_resp
			}
			var reply GetMetadataForSyncReply
			srv.getMetadataForSync(args, &reply)
			returned = reply
		case FCODE_GET_METADATA_OF:
			var args GetMetadataOfArgs
			err := msgpack.Unmarshal(argsBytes, &args)
			if err != nil {
				errCode = ERRCODE_TYPE_ARGUMENT
				goto send_resp
			}
			var reply GetMetadataForSyncReply
			srv.getMetadataOf(args, &reply)
			returned = reply
		default:
			returned = nil
			errCode = ERRCODE_NOFUNCT
		}

	send_resp:
		response := RpcResponse{
			ErrCode: uint8(errCode),
			Id:      request.Id,
		}

		responseBytes, err := msgpack.Marshal(response)
		if err != nil {
			logging.LogTrace(err.Error())
		}
		replyBytes, err := msgpack.Marshal(returned)
		if err != nil {
			logging.LogTrace(err.Error())
		}

		err = compressAndSend(append(responseBytes, replyBytes...), conn)
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

func (rn *NodeClient) Ping(peer Peer) (error, error) {
	var reply PingReply
	err := rn.callRemoteFunction(FCODE_HANDSHAKE, PingArgs{peer}, &reply)
	if err != nil {
		return nil, err
	}
	return reply.Error, nil
}

func (rn *NodeClient) GetStatus() (uint64, uint64, error) {
	var reply GetStatusReply
	err := rn.callRemoteFunction(FCODE_GETSTATUS, GetStatusArgs{}, &reply)
	if err != nil {
		return 0, 0, nil
	}
	return reply.LastTimestamp, reply.LastMetaTimestamp, nil
}

// the LocalNode rpc method equivalent
// Note: style is Function(proxySetting ProxySetting, args) // Proxy settings are mandatory as first argument!!!
func (rn *NodeClient) GetResultsForSync(ts uint64) ([]SearchResult, error) {
	var reply GetResultsForSyncReply
	err := rn.callRemoteFunction(FCODE_GET_RESULTS_FOR_SYNC, TimestampArgs{ts}, &reply)
	if err != nil {
		return nil, err
	}
	return reply.Results, nil
}

func (rn *NodeClient) GetMetadataForSync(ts uint64) ([]ResultMeta, error) {
	var reply GetMetadataForSyncReply
	err := rn.callRemoteFunction(FCODE_GET_METADATA_FOR_SYNC, TimestampArgs{ts}, &reply)
	if err != nil {
		return nil, err
	}
	return reply.Metadatas, nil
}

func (rn *NodeClient) GetPeersForSync() ([]Peer, error) {
	var reply GetPeersForSyncReply
	err := rn.callRemoteFunction(FCODE_GET_PEERS_FOR_SYNC, GetPeersForSyncArgs{}, &reply)
	if err != nil {
		return nil, err
	}
	return reply.Peers, nil
}

func (rn *NodeClient) GetMetadataOf(hashes []Hash256) ([]ResultMeta, error) {
	var reply GetMetadataForSyncReply
	err := rn.callRemoteFunction(FCODE_GET_METADATA_OF, []any{hashes}, &reply)
	if err != nil {
		return nil, err
	}
	return reply.Metadatas, nil
}

func (rn *NodeClient) callRemoteFunction(funCode byte, args interface{}, reply interface{}) error {
	conn, err := rn.proxySettings.NewConnection(rn.peer.Address)
	if err != nil {
		logging.LogTrace("connection error")
		return err
	}

	request := RpcRequest{
		FuncCode: funCode,
		Id:       util.GenerateId12_Str(),
	}

	reqBytes, err := msgpack.Marshal(request)
	if err != nil {
		return err
	}
	argsBytes, err := msgpack.Marshal(args)
	if err != nil {
		return err
	}

	err = compressAndSend(append(reqBytes, argsBytes...), conn)
	if err != nil {
		return err
	}

	recvBytes, _, err := receiveAndDecompress(conn)
	if err != nil {
		conn.Close()
		return err
	}
	conn.Close()

	if len(recvBytes) <= RPC_RESP_BYTESIZE {
		return errors.New("rpc error: malformed reply")
	}

	var response RpcResponse
	err = msgpack.Unmarshal(recvBytes[:RPC_RESP_BYTESIZE], &response)
	if err != nil {
		return err
	}

	if response.Id != request.Id {
		return errors.New("rpc error: the request and response id did not match")
	}

	if response.ErrCode != ERRCODE_NULL {
		return errCodeToError(request.FuncCode, response.ErrCode)
	}

	err = msgpack.Unmarshal(recvBytes[RPC_RESP_BYTESIZE:], reply)
	if err != nil {
		return err
	}
	return nil
}
