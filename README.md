# sniffDogSniff

Sniff Dog Sniff is a customizable, decentralized search engine
* further info coming soon...
* project is a work in progress is not ready for production environment

### Install dependencies
```
pip3 install -r requirements.txt
```

### How to use
```bash
python3 sniffdogsniff.py -c <path_to_config_file>
```


### The config file (config.ini)
```ini
[general]
web_service_http_port = 4000
searches_database_path = ./custom_path.db
peer_to_peer_port = 4222
peer_database_path = ./peers.db
minimum_search_results_threshold = 10
# sync frequency in seconds
peer_sync_frequency = 10

[proxy]
;used for Peer communication
;can be socks5 or http
;proxy_type = socks5
;proxy_host = 127.0.0.1
;proxy_port = 9050

[known_peers]
peer1 = http://free.sniffdogsniff.net:4222
peer2 = http://sniffdogsniff.mirror1:4222
peer3 = http://sniffdogsniff.mirror2:4222

;search engines

[google]
name = Google
user_agent = Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/102.0.0.0 Safari/537.36
search_query_url = https://www.google.com/search?q=
results_container_filter = div.g
result_url_filter = //a/@href
result_title_filter = //h3/text()
```

### Code contribution
If you want to contribute the code you are Welcome!

#### Coding style
* the class attributes must be named with _ before name, e.g. **self**.__name_
* the "private" methods should be named with _ before name, e.g. __do_something(self):_
* leave comments in the code and let us know what the piece of code you have written does
* do not commit .db files please!
* feel free to add comment in some part of the code that are uncommented

#### Special thanks
* https://github.com/joshmarshall/jsonrpctcp for json-rpc code base for the sdsjsonrpc 
