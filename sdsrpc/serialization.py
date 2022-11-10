from sds.search_result import SearchResult
from sds.peers_db import Peer
import msgpack


def serialize(obj):
    return msgpack.dumps(obj, default=_serialize_ext_object)


def _serialize_ext_object(obj):
    if isinstance(obj, SearchResult):
        return msgpack.ExtType(1, msgpack.packb((
            bytes.fromhex(obj.hash),
            obj.title,
            obj.url,
            obj.description,
            obj.content_type,
            obj.score
        )))
    elif isinstance(obj, Peer):
        return msgpack.ExtType(2, msgpack.packb((
            obj.address,
            obj.rank,
            obj.proxy_type,
            obj.proxy_address
        )))


def deserialize(raw_data):
    return msgpack.loads(raw_data, ext_hook=_deserialize_ext_object)


def _deserialize_ext_object(code, raw_data):
    if code == 1:
        b_hash, title, url, desc, ct, score = msgpack.unpackb(raw_data)
        return SearchResult(
            hash=b_hash.hex(),
            title=title,
            url=url,
            description=desc,
            content_type=ct,
            score=score
        )
    elif code == 2:
        addr, rank, pt, p_addr = msgpack.unpackb(raw_data)
        return Peer(
            address=addr,
            rank=rank,
            proxy_type=pt,
            proxy_address=p_addr
        )
