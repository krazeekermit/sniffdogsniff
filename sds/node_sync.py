from threading import Thread, Event
import logging
import time

from sdsrpc.server import RpcTcpServer

from sds.node import LocalNode, RemoteNode


class NodeSyncServer(Thread):
    def __init__(self, local_node: LocalNode):
        Thread.__init__(self, name='Sync Server')
        self._logger = logging.getLogger(self.name)
        self._bind_port = local_node.configs.peer_to_peer_port
        self._server = RpcTcpServer(local_node)

    def run(self):
        self._logger.info(f'Listening on port {self._bind_port}')
        self._server.serve('127.0.0.1', self._bind_port)
        self._logger.info(f'done!')

    def stop_server(self):
        self._logger.info(f'Shutting down...')
        self._server.shutdown()


class NodeSyncWorker(Thread):
    def __init__(self, local_node: LocalNode):
        Thread.__init__(self, name='Syncing Client')
        self._logger = logging.getLogger(name=self.name)
        self._local_node = local_node
        self._sync_freq = local_node.configs.peer_sync_frequency
        self._self_peer = local_node.configs.self_peer
        self._discoverability = local_node.configs.node_discoverable
        self._stop_event = Event()

    def run(self) -> None:
        self._logger.info('Started Sync Client...')
        while not self._stop_event.is_set():
            time.sleep(self._sync_freq)
            self._sync_with_other_peers()

        self._logger.info('done!')

    def stop_client(self):
        self._logger.info('Shutting down...')
        self._stop_event.set()

    def _sync_with_other_peers(self):
        """
        Syncs search results, peers from the remote peer, when the remote peer is unresponsive
        the ranking increases of 1000
        !!! lower ranking highest speed !!!
        :return: None
        """
        peers_list = self._local_node.get_peers()
        hashes = self._local_node.get_hashes()

        s_time = 0
        for p_info in peers_list[:7]:
            self._logger.info(f'Syncing from {p_info.address}')

            try:
                remote_node = RemoteNode(p_info)
                if self._discoverability:
                    remote_node.handshake(self._self_peer)
                s_time = time.time()
                self._local_node.sync_searches_db_from(remote_node.get_results_for_sync(hashes))
                self._local_node.sync_peers_db_from(remote_node.get_peers_for_sync())
            except Exception as ex:
                self._logger.error(f'{ex.__str__()}')
            finally:
                p_info.rank = int((time.time() - s_time) * 1000)
