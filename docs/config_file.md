## Config file (ini)

```ini
search_database_path = ./searches.db              # Path of the Searches database
peers_database_path = ./peers.db                  # Path of the Peers database
web_service_bind_address = 0.0.0.0:8081           # Web Ui bind address
min_search_results_threshold = 50                 # minimum results threshold for sds to search on other engines (if there is no 50 results found in the database then sds search on centralized engines)

known_peers = peer1, peer2 # name of the peers sections
external_search_engines = startpage, yandex, onesearch, brave # name of the engines sections
```

#### Proxy Settings
```ini
[proxy_settings]
i2p_socks5_proxy = addr:port
tor_socks5_proxy = 127.0.0.1:9050
```

* *i2p_socks5_proxy* is the address of the i2p or i2pd daemon socks proxy on your machine
* *tor_socks5_proxy* is the address of the tor daemon socks proxy on your machine

```ini
[node_service]
enabled = yes
create_hidden_service = no
# Note: sniffdogsniff only supports auto-creation of a Tor Hidden Service, I2P must be done manually
# tor_control_port = 9051
bind_address = 0.0.0.0:4111
proxy_type = none
```

If enabled allows other nodes to sync with your node. If you want to let the TOR Hidden service to be created automatically set *create_hidden_service* to yes.
Only TOR Hidden Service creation is supported! If you use for xample I2P you need to setup your address manually. In case you use I2P you need to specify the *bind_address* as i2paddress_yadayadayada.b32.i2p and set the *proxy_type* to i2p. In case you use the hidden service auto creation you do not need to specify the proxy type.