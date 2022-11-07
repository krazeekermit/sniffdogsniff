import socket

from sds.configs import NodeConfigurations
from sds.peers_db import PeersDB, Peer
from sds.local_db import LocalSearchDatabase, SearchResult
from sds.sniffingdog import SniffingDog

from threading import Thread, Lock, Event
import logging
import time
from sds.utils import string_to_host_port_tuple, string_to_proxy_type

from sdsjsonrpc.server import Server
from sdsjsonrpc.errors import ProtocolError
from sdsjsonrpc.client import Client


class NodeManager:
    def __init__(self, configs: NodeConfigurations):
        self._lock = Lock()
        self._configs = configs
        self._local_db = LocalSearchDatabase(configs.searches_db_path)
        self._peers_db = PeersDB(configs.peer_db_path, configs.known_peers_dict)

        self._sniffing_dog = SniffingDog(configs.search_engines, self._local_db,
                                         configs.minimum_search_results_threshold)

    def insert_new_search_result(self, title: str, url: str, description: str, content_type: str):
        self.unlock()
        res = SearchResult(title=title, url=url, description=description, content_type=content_type)
        self._local_db.sync({res.hash: res})
        self.lock()

    def unlock(self):
        self._lock.acquire()

    def lock(self):
        self._lock.release()

    def request_node_searches(self, hashes: list) -> dict:
        searches = dict()
        for h, rs in self._local_db.get_searches().items():
            logging.debug(f'Answering search request -> sync hash: {h}')
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
            self._peers_db.update_peer_rank(peer)
        finally:
            self._lock.release()

    def search(self, text: str, filter_content_types=[]):
        return self._sniffing_dog.do_search(text, filter_content_types)

    @property
    def configs(self):
        return self._configs


class NodeRpcServer(Thread):

    def __init__(self, node_manger: NodeManager):
        Thread.__init__(self, name='NodeRpcServer')

        self._server = Server(('localhost', node_manger.configs.peer_to_peer_port))
        self._server.serializer.register_object(SearchResult, ['hash', 'title', 'url', 'description', 'content_type',
                                                               'score'])
        self._server.serializer.register_object(Peer, ['address', 'rank', 'proxy_type', 'proxy_address'])
        self._server.add_handler(self.request_node_searches_db_data, 'request_node_searches_db_data')
        self._server.add_handler(self.request_node_peers_db_data, 'request_node_peers_db_data')
        self._server.add_handler(self.handshake, 'handshake')

        self._logger = logging.getLogger(name=self.name)
        self._node_manager = node_manger

    def run(self):
        self._logger.info('Starting Rpc Server...')
        self._server.serve()

    def stop_server(self):
        self._logger.info("Stopping Rpc Server...")
        self._server.shutdown()

    def handshake(self, requesting_peer: Peer):
        """
        If the requesting peer is a discoverable peer it will send a rpc request notify_availability
        to the request handling peer, and the handling peer will register it as a suitable
        candidate for future syncing (in peers_db)
        :param requesting_peer: the requesting peer
        :return: None
        """
        self._node_manager.sync_peers_db_from([requesting_peer])

    def request_node_searches_db_data(self, hashes: list):
        """
        Requests to the request handling peer NodeManager the searches stored in its database.
        The result will be only the searches that the requesting peer doesn't have
        :param hashes: the search hashes that the requesting peers already has in its local_db
        :return: dict[hash, SearchResult] dictionary containing hashes and search results for the
                requesting peer if the requesting peer already has these searches in its local_db
                the returning dict of searches will be empty
        """
        self._node_manager.unlock()
        data = self._node_manager.request_node_searches(hashes)
        self._logger.debug(f'rpc_request = request_node_searches_db_data nsr={len(data)}')
        self._node_manager.lock()
        return data

    def request_node_peers_db_data(self):
        """
        Request peers to the handling request peer
        :return: List of peers that the request handling peer has in its peers_db
        """
        self._logger.debug('rpc_request = request_node_peers_db_data')
        self._node_manager.unlock()
        data = self._node_manager.get_peers()
        self._node_manager.lock()
        return data


class PeerSyncClient(Thread):
    def __init__(self, node_manager: NodeManager):
        Thread.__init__(self, name='Syncing Client')
        self._logger = logging.getLogger(name=self.name)
        self._node_manager = node_manager
        self._sync_freq = node_manager.configs.peer_sync_frequency
        self._self_peer = node_manager.configs.self_peer
        self._discoverability = node_manager.configs.node_discoverable
        self._stop_event = Event()

    def run(self) -> None:
        self._logger.info('Started Sync Client...')
        while not self._stop_event.is_set():
            time.sleep(self._sync_freq)
            self._sync_with_other_peers()

        self._logger.info('done!')

    def stop_client(self):
        self._logger.info('Stopping...')
        self._stop_event.set()

    def _sync_with_other_peers(self):
        """
        Syncs search results, peers from the remote peer, when the remote peer is unresponsive
        the ranking increases of 1000
        !!! lower ranking highest speed !!!
        :return: None
        """
        self._node_manager.unlock()
        peers_list = self._node_manager.get_peers()
        hashes = self._node_manager.get_hashes()
        self._node_manager.lock()
        s_time = 0
        for p in peers_list[:7]:
            self._logger.info(f'Syncing from {p.address}')

            try:
                client = self._get_client_for(p)
                if self._discoverability:
                    client.handshake(self._self_peer)
                s_time = time.time()
                self._node_manager.sync_searches_db_from(client.request_node_searches_db_data(hashes))
                self._node_manager.sync_peers_db_from(client.request_node_peers_db_data)
            except socket.error:
                self._logger.error(f'Socket error: problems communicating with {p.address}')
                p.rank += 1000
            except ProtocolError as e:
                self._logger.error(e.message)
            finally:
                p.rank = int((time.time() - s_time) * 1000)
            self._node_manager.update_peer_rank(p)

    @staticmethod
    def _get_client_for(p: Peer):
        client = Client(string_to_host_port_tuple(p.address), key=None)
        if p.has_proxy():
            client.set_proxy(string_to_proxy_type(p.proxy_type), string_to_host_port_tuple(p.proxy_address))
        client.serializer.register_object(SearchResult, ['hash', 'title', 'url', 'description', 'content_type',
                                                         'score'])
        client.serializer.register_object(Peer, ['address', 'rank', 'proxy_type', 'proxy_address'])
        return client

