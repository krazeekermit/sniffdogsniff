import json
import requests
import os


class SdsPeers:
    def __init__(self, configs: dict):
        self._config = configs['peers']
        self._peer_dict = self._load_peers()
        self._peer_list = self._gen_peer_list()

    def _load_peers(self) -> dict:
        peer_file = self._config['peer_db_path'];
        if os.path.exists(peer_file):
            with open(peer_file, 'r') as fi:
                return json.load(fi)
        else:
            self._peer_dict = {'peers': {}}
            self._save_peers()

    def _save_peers(self):
        with open(self._config['peer_db_path'], 'w') as fo:
            json.dump(self._peer_dict, fo)

    def _gen_peer_list(self) -> list:
        peer_list = list()
        for p in self._peer_dict['peers'].keys():
            peer_list.append(p)
        return peer_list

    def _merge_peer_dict(self, dictionary: dict):
        for p in dictionary['peers'].keys():
            entry = dictionary['peers'][p]
            if p not in self._peer_list:
                self._peer_dict['peers'].append(entry)

    def _sync_peers(self):
        for p_url in self._peer_list:
            resp = requests.post(f'{p_url}/api/sync_peers')
            self._merge_peer_dict(json.loads(resp.content))

    def set_rank(self, p: str, rank: int):
        self._peer_dict['peers'][p]['metric'] = rank

    @property
    def get_peer_dict(self) -> dict:
        return self._peer_dict

    @property
    def get_peer_list(self):
        return self._peer_list


