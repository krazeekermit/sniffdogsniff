import select
import zlib
from threading import Thread, Lock
import socket
from collections import deque
from sdsrpc import serialization, request_code
from sdsrpc.dispatcher import RequestDispatcher
from sdsrpc import RECV_CHUNK_LEN


class ClientsHandler(Thread):
    def __init__(self, dispatcher: RequestDispatcher):
        Thread.__init__(self)
        self._lock = Lock()
        self._clients_queue = deque()
        self._dispatcher = dispatcher
        self._keep_alive = True

    def put(self, client_socket: socket.socket):
        self._lock.acquire()
        self._clients_queue.appendleft(client_socket)
        self._lock.release()

    def run(self):
        while self._keep_alive:
            if len(self._clients_queue) > 0:
                self._lock.acquire()
                client_socket = self._clients_queue.pop()
                self._lock.release()

                buffer = b''
                while True:
                    try:
                        b_chunk = client_socket.recv(RECV_CHUNK_LEN)
                    except socket.timeout:
                        break
                    if not b_chunk:
                        break
                    buffer += b_chunk
                    if len(b_chunk) < RECV_CHUNK_LEN:
                        break
                op, fun_code, args = serialization.deserialize(zlib.decompress(buffer))
                if op == request_code.CALL_CODE:
                    try:
                        response = request_code.RETURN_CODE, fun_code, self._dispatcher.dispatch(fun_code, args)
                    except KeyError as ex:
                        response = request_code.ERROR_CODE, fun_code, f'Function {fun_code} not exists: {str(ex)}'
                    except Exception as ex:
                        response = request_code.ERROR_CODE, fun_code, f'Function {fun_code}: {str(ex)}'
                    finally:
                        client_socket.send(zlib.compress(serialization.serialize(response)))

                client_socket.close()

    def stop_handler(self):
        self._keep_alive = False


class RpcTcpServer:
    def __init__(self, dispatcher: RequestDispatcher):
        self._keep_alive = True
        self._clients_handler = ClientsHandler(dispatcher)
        self._server_socket = socket.socket(socket.AF_INET, socket.SOCK_STREAM)

    def serve(self, host: str, port: int):
        if not self._clients_handler.is_alive():
            self._clients_handler.start()

        self._server_socket.setblocking(0)
        self._server_socket.bind((host, port))
        self._server_socket.listen(50)
        self._loop()

        self._clients_handler.stop_handler()
        self._clients_handler.join()

    def _loop(self):
        inputs = [self._server_socket]

        while self._keep_alive:
            readable, writable, exceptional = select.select(inputs, [], [], 1)
            for s in readable:
                if s is self._server_socket:
                    connection, client_address = s.accept()
                    self._clients_handler.put(connection)

    def shutdown(self):
        self._keep_alive = False


