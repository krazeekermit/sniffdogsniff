## Config file (ini)

```ini
work_dir_path = ./ # work dir path usually is /var/sniffdogsniff
log_to_file = no
log_file_name = sds.log
search_database_max_ram_cache_size = 512M # max cache in ram before flush to disk
allow_results_invalidation = no 
min_search_results_threshold = 50                 # minimum results threshold for sds to search on other engines (if there is no 50 results found in the database then sds search on centralized engines)
```

### Initial Peers
```ini
[peer]
address = 127.0.0.1:4222
id = 199ee773164f7a8bedcbef3afa1bf398cf5febeb

[peer]
address = somepeer.onion
id = 1ec6c7a7ec625077c958e9a321ec7f20c7725e70
```
The initial peers to let you jump into the p2p network. Sniffdogsniff uses kademlia
so the id of a peer is required. Usually is the sha1 of the address.
Can be added as much initial peers as wanted by adding peer section multiple times.

#### Proxy Settings
```ini
[proxy_settings]
i2p_socks5_proxy = addr:port
tor_socks5_proxy = 127.0.0.1:9050
```

* *i2p_socks5_proxy* is the address of the i2p or i2pd daemon socks proxy on your machine
* *tor_socks5_proxy* is the address of the tor daemon socks proxy on your machine

### Node RPC p2p server
```ini
[node_service]
enabled = yes # node is visible to other nodes
bind_address = 0.0.0.0:4222
```
If enabled allows other nodes to sync with your node. Bind address specify the p2p
bind address. If using tor or i2p service auto creation bind_address key is not 
needed.

* Example of configuration for tor onionservice listen address auto configuration
```ini
[node_service]
enabled = yes
hidden_service = tor
tor_control_port = 9051
tor_control_auth_password = password
```

* Example of configuration for i2p hiddenservice listen address auto configuration
```ini
[node_service]
enabled = yes # node is visible to other nodes
hidden_service = i2p
i2p_sam_port = 7656
i2p_sam_user = user
i2p_sam_password = password
```

### Web Ui
```ini
[web_ui]
bind_address = 0.0.0.0:8081 # bind address for web server
```

