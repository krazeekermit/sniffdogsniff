package core

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/sniffdogsniff/hiddenservice"
	"github.com/sniffdogsniff/kademlia"
	"github.com/sniffdogsniff/logging"
	"github.com/sniffdogsniff/proxies"
	"github.com/sniffdogsniff/util"
	"github.com/vmihailenco/msgpack/v5"
)

const BUFFER_SIZE = 256
const MAX_THREAD_POOL_SIZE = 1

const NODE_SERVER = "nodeserver"

const (
	FCODE_HANDSHAKE    = 100
	FCODE_FIND_NODE    = 101
	FCODE_STORE_RESULT = 102
	FCODE_FIND_RESULTS = 103
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
	PingerId      kademlia.KadId
	PingerAddress string
}

type PingReply struct {
	Error error
}

func (srv *NodeServer) ping(args PingArgs, reply *PingReply) {
	if !srv.node.CheckNode(args.PingerId, args.PingerAddress) {
		return
	}
	(*reply).Error = srv.node.Ping(args.PingerId, args.PingerAddress)
}

type FindNodeArgs struct {
	SourceNodeId      kademlia.KadId
	SourceNodeAddress string

	TargetNodeId kademlia.KadId
}

type FindNodeReply struct {
	NewNodes map[kademlia.KadId]string
}

func (srv *NodeServer) findNode(args FindNodeArgs, reply *FindNodeReply) {
	if !srv.node.CheckNode(args.SourceNodeId, args.SourceNodeAddress) {
		return
	}

	srv.node.NodeConnected(args.SourceNodeId, args.SourceNodeAddress)
	(*reply).NewNodes = srv.node.FindNode(args.TargetNodeId)
}

type StoreResultArgs struct {
	SourceNodeId      kademlia.KadId
	SourceNodeAddress string

	Value SearchResult
}

type StoreResultReply struct {
}

func (srv *NodeServer) storeResult(args StoreResultArgs, reply *StoreResultReply) {
	if !srv.node.CheckNode(args.SourceNodeId, args.SourceNodeAddress) {
		return
	}

	srv.node.NodeConnected(args.SourceNodeId, args.SourceNodeAddress)
	srv.node.StoreResult(args.Value)
}

type FindResultsArgs struct {
	SourceNodeId      kademlia.KadId
	SourceNodeAddress string

	QueryString string
}

type FindResultsReply struct {
	Values []SearchResult
}

func (srv *NodeServer) findResults(args FindResultsArgs, reply *FindResultsReply) {
	if !srv.node.CheckNode(args.SourceNodeId, args.SourceNodeAddress) {
		return
	}

	srv.node.NodeConnected(args.SourceNodeId, args.SourceNodeAddress)
	(*reply).Values = srv.node.FindResults(args.QueryString)
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
		logging.Errorf(NODE_SERVER, err.Error())
		return
	}
	logging.Infof(NODE_SERVER, "NodeServer is listening on %s", proto.GetAddressString())

	go srv.acceptConns(listener)
}

func (srv *NodeServer) acceptConns(listener net.Listener) {
	defer listener.Close()
	for {
		conn, err := listener.Accept()
		logging.Debugf(NODE_SERVER, "New connection from %s", conn.RemoteAddr().String())
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
			logging.Errorf(NODE_SERVER, err.Error())
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
			logging.Debugf(NODE_SERVER, "Malformed rpc request")
			errCode = ERRCODE_MARSHAL
			goto send_resp
		}
		argsBytes = recvBytez[RPC_REQ_BYTESIZE:]

		logging.Debugf(NODE_SERVER, "Function request %d size %d", request.FuncCode, len(recvBytez))

		switch request.FuncCode {
		case FCODE_HANDSHAKE:
			var args PingArgs
			err := msgpack.Unmarshal(argsBytes, &args)
			if err != nil {
				logging.Debugf(NODE_SERVER, err.Error())
				errCode = ERRCODE_TYPE_ARGUMENT
				goto send_resp
			}
			var reply PingReply
			srv.ping(args, &reply)
			returned = reply
		case FCODE_FIND_NODE:
			var args FindNodeArgs
			err := msgpack.Unmarshal(argsBytes, &args)
			if err != nil {
				logging.Debugf(NODE_SERVER, err.Error())
				errCode = ERRCODE_TYPE_ARGUMENT
				goto send_resp
			}
			var reply FindNodeReply
			srv.findNode(args, &reply)
			returned = reply
		case FCODE_STORE_RESULT:
			var args StoreResultArgs
			err := msgpack.Unmarshal(argsBytes, &args)
			if err != nil {
				errCode = ERRCODE_TYPE_ARGUMENT
				goto send_resp
			}
			var reply StoreResultReply
			srv.storeResult(args, &reply)
			returned = reply
		case FCODE_FIND_RESULTS:
			var args FindResultsArgs
			err := msgpack.Unmarshal(argsBytes, &args)
			if err != nil {
				errCode = ERRCODE_TYPE_ARGUMENT
				goto send_resp
			}
			var reply FindResultsReply
			srv.findResults(args, &reply)
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
			logging.Debugf(NODE_SERVER, err.Error())
		}
		replyBytes, err := msgpack.Marshal(returned)
		if err != nil {
			logging.Debugf(NODE_SERVER, err.Error())
		}

		err = compressAndSend(append(responseBytes, replyBytes...), conn)
		if err != nil {
			logging.Debugf(NODE_SERVER, err.Error())
		}
		conn.Close()
	}
}

type NodeClient struct {
	addr          string
	proxySettings proxies.ProxySettings
}

func NewNodeClient(addr string, proxySettings proxies.ProxySettings) NodeClient {
	return NodeClient{
		addr:          addr,
		proxySettings: proxySettings,
	}
}

/***************************** Remote Node (Client) ******************************/

func (rn *NodeClient) Ping(id kademlia.KadId, addr string) (error, error) {
	var reply PingReply
	err := rn.callRemoteFunction(FCODE_HANDSHAKE, PingArgs{id, addr}, &reply)
	if err != nil {
		return nil, err
	}
	return reply.Error, nil
}

func (rn *NodeClient) FindNode(targetId kademlia.KadId, source *kademlia.KNode) (map[kademlia.KadId]string, error) {
	var reply FindNodeReply
	err := rn.callRemoteFunction(
		FCODE_FIND_NODE,
		FindNodeArgs{TargetNodeId: targetId, SourceNodeId: source.Id, SourceNodeAddress: source.Address},
		&reply,
	)
	if err != nil {
		return nil, err
	}
	return reply.NewNodes, nil
}

func (rn *NodeClient) StoreResult(sr SearchResult, source *kademlia.KNode) error {
	var reply StoreResultReply
	err := rn.callRemoteFunction(
		FCODE_STORE_RESULT,
		StoreResultArgs{Value: sr, SourceNodeId: source.Id, SourceNodeAddress: source.Address},
		&reply,
	)
	if err != nil {
		return err
	}
	return nil
}

func (rn *NodeClient) FindResults(query string, source *kademlia.KNode) ([]SearchResult, error) {
	var reply FindResultsReply
	err := rn.callRemoteFunction(
		FCODE_FIND_RESULTS,
		FindResultsArgs{QueryString: query, SourceNodeId: source.Id, SourceNodeAddress: source.Address},
		&reply,
	)
	if err != nil {
		return nil, err
	}
	return reply.Values, nil
}

func (rn *NodeClient) callRemoteFunction(funCode byte, args interface{}, reply interface{}) error {
	conn, err := rn.proxySettings.NewConnection(rn.addr)
	if err != nil {
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
