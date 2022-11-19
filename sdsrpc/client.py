import time
import zlib

from sdsrpc import RECV_CHUNK_LEN, BYTES_TO_MB_CONV
from sdsrpc import request_code
from sdsrpc import serialization
from sdsrpc.exceptions import RpcRequestException, RpcClientConnectionException
import socks
import socket


class RpcTcpClient:
    def __init__(self, host: str, port: int, proxy_type: int, proxy_host: str, proxy_port: int):
        self._host = host
        self._port = port
        self._proxy_type = proxy_type
        self._proxy_host = proxy_host
        self._proxy_port = proxy_port

        self._last_request_dl_speed = 0

    def use_proxy(self):
        return self._proxy_type != -1

    def call_remote(self, fun_code: int, *args):
        return self._connect_and_perform_request((request_code.CALL_CODE, fun_code, args))

    def _connect_and_perform_request(self, request_data):
        client_socket = socks.socksocket()
        if self.use_proxy():
            client_socket.set_proxy(self._proxy_type, addr=self._proxy_host, port=self._proxy_port)

        try:
            client_socket.connect((self._host, self._port))
            client_socket.send(zlib.compress(serialization.serialize(request_data)))
        except socket.error as ex:
            raise RpcClientConnectionException(self._host, str(ex))

        start_time = 0

        buffer = b''
        while True:
            try:
                b_chunk = client_socket.recv(RECV_CHUNK_LEN)
                start_time = time.time()
            except client_socket.timeout:
                break
            if not b_chunk:
                break
            buffer += b_chunk
            if len(b_chunk) < RECV_CHUNK_LEN:
                break

        elapsed_time = (time.time() - start_time) * 1000  # download time in seconds
        uncompressed_data = zlib.decompress(buffer)
        self._last_request_dl_speed = int(((len(uncompressed_data) - RECV_CHUNK_LEN) / elapsed_time) / BYTES_TO_MB_CONV)

        op, fun_code, ret = serialization.deserialize(uncompressed_data)
        if op == request_code.RETURN_CODE:
            return ret
        else:
            raise RpcRequestException(ret, fun_code)

    def get_last_request_download_speed(self):
        """
        :return: download speed in MB/s (approx)
        """
        return self._last_request_dl_speed
