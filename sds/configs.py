from configparser import ConfigParser
import os
from sds.sniffingdog import SearchEngine
from sds.peers_db import Peer

sections_to_exclude = ['general', 'proxy', 'known_peers']
general_section_keys = ['web_service_http_port', 'searches_database_path', 'minimum_search_results_threshold',
                        'peer_to_peer_port', 'peer_database_path', 'peer_sync_frequency']


class NodeConfigurations:

    def __init__(self):
        self._config_parser = ConfigParser(allow_no_value=True)
        self.__general_configs = dict()
        self._search_engines = list()
        self._known_peers = dict()

    def read_from_file(self, file_path: str):
        self._config_parser.read(file_path)
        self.__general_configs = dict(self._config_parser['general'])
        self._parse()

    def read_from_env_variables(self):
        for k in general_section_keys:
            val = os.environ.get(k.upper())
            if val is not None:
                self.__general_configs[k] = val

    def _parse(self):
        if self._config_parser['general']['engines'] is not None:
            for sn in self._config_parser['general']['engines'].split(','):
                sec = self._config_parser[sn.strip()]
                engine = SearchEngine(sec['name'], sec['search_query_url'], sec['results_container_filter'],
                                      sec['result_url_filter'], sec['result_title_filter'], sec['user_agent'])
            self._search_engines.append(engine)

        if self._config_parser['general']['peers'] is not None:
            for pn in self._config_parser['general']['peers'].split(','):
                sec = self._config_parser[pn.strip()]
                proxy_type = sec['proxy_type']
                proxy_addr = None
                if proxy_type != 'none':
                    proxy_addr = sec['proxy_address']
                self._known_peers[pn] = Peer(address=sec['address'], rank=0, proxy_type=proxy_type,
                                             proxy_address=proxy_addr)

    @property
    def search_engines(self) -> list:
        return self._search_engines

    @property
    def web_service_http_port(self):
        return self.__general_configs['web_service_http_port']

    @property
    def searches_db_path(self) -> str:
        return self.__general_configs['searches_database_path']

    @property
    def minimum_search_results_threshold(self) -> int:
        return int(self.__general_configs['minimum_search_results_threshold'])

    @property
    def peer_to_peer_port(self):
        return int(self.__general_configs['peer_to_peer_port'])

    @property
    def peer_db_path(self):
        return self.__general_configs['peer_database_path']

    @property
    def known_peers_dict(self) -> dict:
        return self._known_peers

    @property
    def peer_sync_frequency(self):
        return int(self.__general_configs['peer_sync_frequency'])
