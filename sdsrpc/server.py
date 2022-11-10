import select
import zlib
from threading import Thread, Lock
import socket
from collections import deque
from sdsrpc import serialization
from sdsrpc.dispatcher import RequestDispatcher


class ClientsHandler(Thread):
    def __init__(self, dispatcher: RequestDispatcher):
        Thread.__init__(self)
        self._lock = Lock()
        self._clients_queue = deque()
        self._dispatcher = dispatcher
        self._stop = False

    def put(self, client_socket: socket.socket):
        self._lock.acquire()
        self._clients_queue.appendleft(client_socket)
        self._lock.release()

    def run(self):
        while not self._stop:
            print(f'queue waiting {len(self._clients_queue)}')
            if len(self._clients_queue) > 0:
                self._lock.acquire()
                client_socket = self._clients_queue.pop()
                self._lock.release()

                buffer = b''
                while True:
                    b_chunk = client_socket.recv(2 * 1024)
                    if not b_chunk:
                        break
                    buffer += b_chunk
                request = serialization.deserialize(zlib.decompress(buffer))
                response = self._dispatcher.dispatch(request)
                client_socket.send(zlib.compress(serialization.serialize(response)))
                client_socket.close()

    def stop_handler(self):
        self._stop = True


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
                    print(f'connection from {client_address}')
                    self._clients_handler.put(connection)

    def shutdown(self):
        self._keep_alive = False


if __name__ == '__main__':
    server = RpcTcpServer(RequestDispatcher())
    server.serve('127.0.0.1', 45002)

