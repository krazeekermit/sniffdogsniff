import json
from socket import socket
import zlib
import msgpack
from sds.search_result import SearchResult
import pickle


def receive_and_unpack_data(client_socket: socket):
    buffer = b''
    while True:
        b_chunk = client_socket.recv(1024)
        print(f'data -> {b_chunk}')
        if not b_chunk:
            break
        buffer += b_chunk
    return msgpack.unpackb(buffer)


def pack_data(data):
    return msgpack.packb(data)


def serialize(obj):
    if isinstance(obj, SearchResult):
        return msgpack.ExtType(1, msgpack.packb((
            bytes.fromhex(obj.hash),
            obj.title,
            obj.url,
            obj.description
        )))


def deserialize(code, s_bytes):
    if code == 1:
        bhash, title, url, desc = msgpack.unpackb(s_bytes)
        return SearchResult(
            hash=bhash.hex(),
            title=title,
            url=url,
            description=desc
        )




if __name__ == '__main__':
    buf = b''
    sr = SearchResult(title='Title', url='www.google.com', description='desc')
    buf += pickle.dumps(['hello', 'wello', {'dictello':False}])
    packed = msgpack.dumps(['hello', 'wello', sr], default=serialize)
    print(f"msgpack {packed}")
    print(msgpack.loads(packed, ext_hook=deserialize))
