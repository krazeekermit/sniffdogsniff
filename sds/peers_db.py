import sqlite3
from os.path import exists


class Peer:

    def __init__(self, **kwargs):
        self._address = kwargs['address']
        self._rank = kwargs['rank']
        self._proxy_type = kwargs['proxy_type']
        self._proxy_address = kwargs['proxy_address']

    def __dict__(self):
        return {'address': self._address, 'rank': self._rank, 'proxy_type': self._proxy_type,
                'proxy_address': self._proxy_address}

    def has_proxy(self) -> bool:
        return self.proxy_type != 'none'

    @property
    def address(self, address):
        self._address = address

    @property
    def rank(self):
        return self._rank

    @rank.setter
    def rank(self, rank: int):
        self._rank = rank

    @property
    def proxy_type(self):
        return self._proxy_type

    @property
    def proxy_address(self):
        return self._proxy_address


class PeersDB:

    def __init__(self, db_path, known_peers: dict):
        self._db_path = db_path
        self._conn = None
        self._check_db_exists()
        self._known_peers = list(known_peers.values())

    def _check_db_exists(self):
        if not exists(self._db_path):
            self._conn = sqlite3.connect(self._db_path, check_same_thread=False)
            self._conn.execute(
                'create table peers(address text, rank int, proxy_type text, proxy_addr text)')
            # self._conn.commit()
        else:
            self._conn = sqlite3.connect(self._db_path, check_same_thread=False)

    def get_peers(self) -> list:
        cur = self._conn.cursor()
        cur.execute('select * from peers')
        peers = list()
        peers.extend(self._known_peers)
        for row in cur.fetchall():
            peers.append(Peer(address=row[0], rank=row[1], proxy_type=row[3], proxy_address=row[4]))
        return peers

    def peers_addresses(self) -> list:
        addresses = list()
        for p in self.get_peers():
            addresses.append(p.address)
        return addresses

    def sync_peers_from(self, peers: list):
        addresses = self.peers_addresses()
        for p in peers:
            if p.address not in addresses:
                self._conn.execute(
                    f'insert into peers values("{p.address}", {p.rank}, "{p.proxy_type}", "{p.proxy_address}")'
                )

        self._conn.commit()

    def update_peer_rank(self, peer: Peer):
        self._conn.execute(
            f'update peers set rank={peer.rank} where address = {peer.address})'
        )
        self._conn.commit()


