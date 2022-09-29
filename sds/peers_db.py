import sqlite3
from os.path import exists


class Peer:

    def __init__(self, **kwargs):
        self._address = kwargs['address']
        self._rank = kwargs['rank']

    def __dict__(self):
        return {'address': self._address, 'rank': self._rank}

    @property
    def address(self, address):
        self._address = address

    @property
    def rank(self):
        return self._rank

    @rank.setter
    def rank(self, rank: int):
        self._rank = rank


class PeersDB:

    def __init__(self, db_path, known_peers: dict):
        self._db_path = db_path
        self._conn = None
        self._check_db_exists()
        self._known_peers = list()
        self._peer_from_configs(known_peers)

    def _peer_from_configs(self, known_peers: dict):
        for pn, pu in known_peers.items():
            self._known_peers.append(Peer(pu, 0))

    def _check_db_exists(self):
        if not exists(self._db_path):
            self._conn = sqlite3.connect(self._db_path, check_same_thread=False)
            self._conn.execute(
                'create table peers(address text, rank int)')
            # self._conn.commit()
        else:
            self._conn = sqlite3.connect(self._db_path, check_same_thread=False)

    def get_peers(self) -> list:
        cur = self._conn.cursor()
        cur.execute('select * from peers')
        peers = list()
        peers.extend(self._known_peers)
        for row in cur.fetchall():
            peers.append(Peer(row[0], row[1], row[2]))
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
                    f'insert into peers values("{p.address}", {p.rank})'
                )

        self._conn.commit()


