from sds.configs import NodeConfigurations
from sds.peers_db import PeersDB, PeerInfo
from sds.local_db import LocalResultsDB, SearchResult
from sds.seeker import SeekerDog

from threading import Lock
import logging
from sds import utils

from sdsrpc.dispatcher import RequestDispatcher
from sdsrpc.client import RpcTcpClient


GET_RESULTS_FOR_SYNC = 101
GET_PEERS_FOR_SYNC = 102
HANDSHAKE = 103


class LocalNode(RequestDispatcher):
    def __init__(self, configs: NodeConfigurations):
        RequestDispatcher.__init__(self)
        self._lock = Lock()
        self._configs = configs
        self._local_db = LocalResultsDB(configs.searches_db_path)
        self._peers_db = PeersDB(configs.peer_db_path, configs.known_peers_dict)

        self._sniffing_dog = SeekerDog(configs.search_engines, self._local_db,
                                       configs.minimum_search_results_threshold)

        # *** configuration of remote callable functions *** #
        self.register_function(HANDSHAKE, self.handshake)
        self.register_function(GET_RESULTS_FOR_SYNC, self.get_results_for_sync)
        self.register_function(GET_PEERS_FOR_SYNC, self.get_peers_for_sync)

    def handshake(self, peer: PeerInfo) -> dict:
        """
        get_results_for_sync: remote callable function
        :param peer: peer information about the requesting peer (if discoverable), if is not already in peerDb it will be
                    added
        :return: void
        """
        self._lock.acquire()
        try:
            self._peers_db.sync_peers_from([peer])
        finally:
            self._lock.release()

    def get_results_for_sync(self, hashes: list) -> dict:
        """
        get_results_for_sync: remote callable function
        :param hashes:
        :return: list of search result that are not in the hash list
                (hashes (remote) - hashes (local) = results to send back)
                retrieved from LocalResultsDB
        """
        self._lock.acquire()
        searches = list()
        for sr in self._local_db.get_searches():
            if sr.hash not in hashes:
                searches.append(sr)

        self._lock.release()
        return searches

    def get_peers_for_sync(self):
        """
        get_peers_for_sync: remote callable function
        :return: the list of peers known by PeersDB
        """
        self._lock.acquire()
        peer_list = []
        peer_list.extend(self.get_peers())
        self._lock.release()
        return peer_list

    def unlock(self):
        self._lock.acquire()

    def lock(self):
        self._lock.release()

    def insert_new_search_result(self, title: str, url: str, description: str, content_type: str):
        self._lock.acquire()
        res = SearchResult(title=title, url=url, description=description, content_type=content_type)
        self._local_db.sync({res.hash: res})
        self._lock.release()

    def update_result_score(self, r_hash, increment: int):
        self._lock.acquire()
        try:
            self._local_db.update_result_score(r_hash, increment)
        finally:
            self._lock.release()

    def get_peers(self) -> list:
        self._lock.acquire()
        peer_list = []
        peer_list.extend(self._peers_db.get_peers())
        self._lock.release()
        return peer_list

    def get_hashes(self):
        self._lock.acquire()
        hashes = []
        hashes.extend(self._local_db.get_hashes())
        self._lock.release()
        return hashes

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

    def update_peer_rank(self, peer: PeerInfo):
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


class RemoteNode:
    def __init__(self, peer_info: PeerInfo):
        self._client = RpcTcpClient(
            utils.host_from_url(peer_info.address),
            utils.port_from_url(peer_info.address),
            peer_info.proxy_type,
            utils.host_from_url(peer_info.proxy_address),
            utils.port_from_url(peer_info.proxy_address)
        )

    def get_download_speed(self):
        return self._client.get_last_request_download_speed()

    def handshake(self, peer: PeerInfo):
        return self._client.call_remote(HANDSHAKE, peer)

    def get_results_for_sync(self, hashes: list) -> dict:
        return self._client.call_remote(GET_RESULTS_FOR_SYNC, hashes)

    def get_peers_for_sync(self):
        return self._client.call_remote(GET_PEERS_FOR_SYNC)


