import json
import time
import threading
import zlib

import msgpack

from sdsrpc import server, client, dispatcher


"""
To be removed after a little bit of play around
"""


def junkmethod():
    for _ in range(0, 9):
        print('Hello')


if __name__ == '__main__':
    disp = dispatcher.RequestDispatcher()
    disp.register_function(101, junkmethod)
    srv = server.RpcTcpServer(disp)
    threading.Thread(target=srv.serve, args=('127.0.0.1', 1234)).start()

    time.sleep(3)
    cli = client.RpcTcpClient('127.0.0.1', 1234, -1, None, None)
    cli.call_remote(101)

    print(f'Server shutdown test')
    srv.shutdown()
