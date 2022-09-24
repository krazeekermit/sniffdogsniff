from .sdsconfigs import SdsConfigs
from .peers_db import PeersDB, Peer
from .local_db import LocalSearchDatabase, SearchResult
from .sniffingdog import SniffingDog
import zmq
from tinyrpc.server import RPCServer
from tinyrpc import RPCClient
from tinyrpc.dispatch import RPCDispatcher
from tinyrpc.protocols.jsonrpc import JSONRPCProtocol
from tinyrpc.transports.zmq import ZmqServerTransport, ZmqClientTransport
from threading import Thread, Lock
import logging
import time


class NodeManager:

    def __init__(self, configs: SdsConfigs):
        self._lock = Lock()
        self._configs = configs
        self._local_db = LocalSearchDatabase(configs.searches_db_path)
        self._peers_db = PeersDB(configs.peer_db_path, configs.known_peers_dict)

        self._sniffing_dog = SniffingDog(configs.search_engines, self._local_db,
                                         configs.minimum_search_results_threshold)

    def insert_new_search_result(self, title: str, url: str, description: str):
        res = SearchResult('', title, url, description)
        self._local_db.sync({res.hash, res})

    def unlock(self):
        self._lock.acquire()

    def lock(self):
        self._lock.release()

    def request_node_searches(self, hashes: list) -> dict:
        searches = dict()
        for h, rs in self._local_db.get_searches().items():
            if h not in hashes:
                searches[h] = rs
        return searches

    def get_peers(self) -> list:
        return self._peers_db.get_peers()

    def get_hashes(self):
        return self._local_db.get_hashes()

    def sync_searches_db_from(self, new_searches: dict):
        self._lock.acquire()
        try:
            self._local_db.sync_from(new_searches)
        finally:
            self._lock.release()

    def sync_peers_db_from(self, new_peers: list):
        self._lock.acquire()
        try:
            self._peers_db.sync_peers_from(new_peers)
        finally:
            self._lock.release()

    def update_peer_rank(self, peer: Peer):
        self._lock.acquire()
        try:
            self._peers_db.update_peer(peer)
        finally:
            self._lock.release()

    @property
    def configs(self):
        return self._configs


class NodeRpcServer(Thread):

    def __init__(self, node_manger: NodeManager):
        Thread.__init__(self, name='NodeRpcServer Thread')
        ctx = zmq.Context()
        dispatcher = RPCDispatcher()
        dispatcher.add_method(self.request_node_searches_db_data, name='request_node_searches_db_data')
        dispatcher.add_method(self.request_node_peers_db_data, name='request_node_peers_db_data')
        transport = ZmqServerTransport.create(
            ctx,
            f'tcp://127.0.0.1:{node_manger.configs.peer_to_peer_port}'
        )
        self._rpc_server = RPCServer(
            transport,
            JSONRPCProtocol(),
            dispatcher
        )

        self._logger = logging.getLogger(name=self.name)
        self._node_manager = node_manger

    def run(self):
        self._logger.info('Starting Rpc Server')
        self._rpc_server.serve_forever()

    def request_node_searches_db_data(self, hashes: list):
        self._logger.debug('rpc_request = request_node_searches_db_data')
        self._node_manager.unlock()
        data = self._node_manager.request_node_searches(hashes)
        self._node_manager.lock()
        return data

    def request_node_peers_db_data(self):
        self._logger.debug('rpc_request = request_node_peers_db_data')
        self._node_manager.unlock()
        data = self._node_manager.get_peers()
        self._node_manager.lock()
        return data


class PeerSyncManager(Thread):

    def __init__(self, node_manager: NodeManager):
        Thread.__init__(self, name='PeerSyncManager Thread')
        self._logger = logging.getLogger(name=self.name)
        self._node_manager = node_manager
        self._sync_freq = node_manager.configs.peer_sync_frequency

    def run(self) -> None:
        while True:
            time.sleep(self._sync_freq)
            self._logger.debug('Syncing...')
            # self._sync_with_other_peers()

    def _sync_with_other_peers(self):
        self._node_manager.unlock()
        peers_list = self._node_manager.get_peers()
        hashes = self._node_manager.get_hashes()
        self._node_manager.lock()
        for p in peers_list:
            logging.info(f'Syncing from {p.name}')
            s_time = time.time()
            client = self._get_client_for_peer(p.address)
            self._node_manager.sync_searches_db_from(client.request_node_searches_db_data(hashes))
            self._node_manager.sync_peers_db_from(client.request_node_peers_db_data)
            p.rank = int((time.time() - s_time) * 1000)
            self._node_manager.update_peer_rank(p)

    @staticmethod
    def _get_client_for_peer(address: str) -> RPCClient:
        ctx = zmq.Context()
        client = RPCClient(
            JSONRPCProtocol(),
            ZmqClientTransport.create(ctx, address)
        )
        return client.get_proxy()


def start_sds_node(node_manager: NodeManager):
    server = NodeRpcServer(node_manager)
    peer_manager = PeerSyncManager(node_manager)
    logging.info('Starting NodeRpcServer')
    server.start()
    logging.info('Starting Peers Sync Manager')
    peer_manager.start()
