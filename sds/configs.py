from configparser import ConfigParser
import os
import logging
from sds.seeker import SearchEngine
from sds.peers_db import PeerInfo
from sds import utils

sections_to_exclude = ['general', 'proxy', 'known_peers']
general_section_keys = ['web_service_http_port', 'searches_database_path', 'minimum_search_results_threshold',
                        'peer_to_peer_port', 'peer_database_path', 'peer_sync_frequency']


class NodeConfigurations:

    def __init__(self):
        self._config_parser = ConfigParser(allow_no_value=True)
        self._general_configs = dict()
        self._node_configs = dict()
        self._search_engines = list()
        self._known_peers = list()

    def read_from_file(self, file_path: str):
        self._config_parser.read(file_path)
        self._general_configs = dict(self._config_parser['general'])
        self._node_configs = dict(self._config_parser['node'])
        self._parse()

    def read_from_env_variables(self):
        for k in general_section_keys:
            val = os.environ.get(k.upper())
            if val is not None:
                self._general_configs[k] = val

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
                self._known_peers.append(PeerInfo(address=sec['address'], rank=0,
                                                  proxy_type=utils.string_to_proxy_type(proxy_type),
                                                  proxy_address=proxy_addr))

    @property
    def search_engines(self) -> list:
        return self._search_engines

    @property
    def web_service_http_port(self):
        return self._general_configs['web_service_http_port']

    @property
    def web_service_http_host(self):
        return self._general_configs['web_service_http_host']

    @property
    def searches_db_path(self) -> str:
        return self._general_configs['searches_database_path']

    @property
    def minimum_search_results_threshold(self) -> int:
        return int(self._general_configs['minimum_search_results_threshold'])

    @property
    def peer_to_peer_port(self):
        return int(self._node_configs['peer_to_peer_port'])

    @property
    def node_discoverable(self) -> bool:
        return self._node_configs['discoverable']

    @property
    def self_peer(self) -> PeerInfo:
        return PeerInfo(
            address=self._node_configs['node_address'],
            proxy_type=utils.string_to_proxy_type(self._node_configs['proxy_type']),
            proxy_address=self._node_configs['proxy_address']
        )

    @property
    def peer_db_path(self):
        return self._general_configs['peer_database_path']

    @property
    def known_peers_dict(self) -> dict:
        return self._known_peers

    @property
    def peer_sync_frequency(self):
        return int(self._general_configs['peer_sync_frequency'])

    @property
    def log_level(self):
        if str(self._general_configs['log_level']).lower() == 'info':
            return logging.INFO
        elif str(self._general_configs['log_level']).lower() == 'warning':
            return logging.WARNING
        elif str(self._general_configs['log_level']).lower() == 'error':
            return logging.ERROR
        elif str(self._general_configs['log_level']).lower() == 'debug':
            return logging.DEBUG
