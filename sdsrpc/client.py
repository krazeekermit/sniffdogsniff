from sdsrpc import request_code
from sdsrpc import serialization
import socks


class RpcTcpClient:
    def __init__(self, host: str, port: int, proxy_type: int, proxy_host: str, proxy_port: int):
        self._host = host
        self._port = port
        self._proxy_type = proxy_type
        self._proxy_host = proxy_host
        self._proxy_port = proxy_port

    def use_proxy(self):
        return self._proxy_type != -1

    def call_remote(self, fun_code: int, *args):
        return self._connect_and_perform_request((request_code.CALL_CODE, fun_code, args))

    def _connect_and_perform_request(self, request_data):
        socket = socks.socksocket()
        
        socket.connect((self._host, self._port))
        return None
