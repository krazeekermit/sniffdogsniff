from configparser import ConfigParser
from .sniffingdog import SearchEngine

sections_to_exclude = ['general', 'proxy', 'known_peers']


class SdsConfigs:

    def __init__(self):
        self._config = ConfigParser()
        self._search_engines = list()
        self._known_peers = dict()

    def read_from_file(self, file_path: str):
        self._config.read(file_path)
        print(self._config.sections())
        for sn in self._config.sections():
            if sn not in sections_to_exclude:
                sec = self._config[sn]
                engine = SearchEngine(sec['name'], sec['search_query_url'], sec['results_container_filter'],
                                      sec['result_url_filter'], sec['result_title_filter'], sec['user_agent'])
                self._search_engines.append(engine)

        for pn in self._config['known_peers'].keys():
            self._known_peers[pn] = self._config['known_peers'][pn]

    @property
    def search_engines(self) -> list:
        return self._search_engines

    @property
    def web_service_http_port(self):
        return self._config['general']['web_service_http_port']

    @property
    def searches_db_path(self) -> str:
        return self._config['general']['searches_database_path']

    @property
    def minimum_search_results_threshold(self) -> int:
        return int(self._config['general']['minimum_search_results_threshold'])

    @property
    def peer_to_peer_port(self):
        return int(self._config['general']['peer_to_peer_port'])

    @property
    def peer_db_path(self):
        return self._config['general']['peer_database_path']

    @property
    def known_peers_dict(self) -> dict:
        return self._known_peers

    @property
    def peer_sync_frequency(self):
        return int(self._config['general']['peer_sync_frequency'])
