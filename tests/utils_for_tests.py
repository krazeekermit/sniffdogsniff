from multiprocessing import Process
from sds.sdsconfigs import SdsConfigs
from sds.node import NodeManager, NodeRpcServer


def _start_test_rpc_server_process(server: NodeRpcServer):
    server.start()
    server.join()


def start_test_rpc_server(node_manager: NodeManager) -> Process:
    server = NodeRpcServer(node_manager)
    proc = Process(target=_start_test_rpc_server_process, args=(server,))
    proc.start()
    return proc
