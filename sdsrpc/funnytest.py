import json
import time
import threading
import zlib
from sds.node import RemoteNode
from sds.peers_db import PeerInfo

import msgpack

from sdsrpc import server, client, dispatcher


"""
To be removed after a little bit of play around
"""


def junkmethod():
    for _ in range(0, 9):
        print('Hello')


def test3():
    print(f' *** Asking searches *** ')
    p = PeerInfo(address='tcp://127.0.0.1:4222', proxy_type='None', proxy_address='')
    rn = RemoteNode(p)
    rr = rn.get_results_for_sync([])
    print(rr)
    hashes = set()
    hashes_ls = list()
    for sr1 in rr:
        hashes.add(sr1.hash)
        hashes_ls.append(sr1.hash)

    print(len(hashes_ls) - len(hashes))
    print(len(rr))


if __name__ == '__main__':
    test3()
