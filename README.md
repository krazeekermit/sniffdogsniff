![SniffDogSniff logo](sniffdogsniff_icon.png "Logo")

# sniffDogSniff

Sniff Dog Sniff is a customizable, decentralized search engine
* further info coming soon...
* project is a work in progress is not ready for production environment

### How to use
```bash
sniffdogsniff --config config.ini
```


### The config file (config.ini)
```ini
search_database_path = ./searches.db
p2p_rpc_port = 4222
peers_database_path = ./peers.db
web_service_bind_address = 0.0.0.0:8081
min_search_results_threshold = 50

known_peers = peer1, peer2
external_search_engines = startpage, yandex, onesearch, brave

[node_service]
enabled = yes
create_hidden_service = no
# tor_control_port = 9051
bind_address = 0.0.0.0:4111
proxy_type = none

[peer1]
address = 127.0.0.1:4222
proxy_type = none

[peer2]
address = 192.178.1.1:4222
proxy_type = socks5
proxy_address = 127.0.0.1:9050

# search engines

[startpage]
name = StartPage
user_agent = Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/102.0.0.0 Safari/537.36
search_query_url = https://www.startpage.com/sp/search?q=%s
results_container_element = div.mainline-results
result_container_element = div.w-gl__result-second-line-container
result_url_element = a.result-link
result_url_property = href
result_title_element = h3
result_title_property = text

[yandex]
name = Yandex
user_agent = Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/102.0.0.0 Safari/537.36
search_query_url = https://yandex.com/search/?text=%s&lr=10448
results_container_element = ul.serp-list
result_container_element = li.serp-item
result_url_element = a.link
result_url_property = href
result_title_element = h2
result_title_property = text

[onesearch]
name = Onesearch
user_agent = Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/102.0.0.0 Safari/537.36
search_query_url = https://www.onesearch.com/yhs/search?q=%s
results_container_element = ol.searchCenterMiddle
result_container_element = div.relsrch
result_url_element = a.ac-algo
result_url_property = href
result_title_element = a.ac-algo
result_title_property = text

[brave]
name = Brave
user_agent = Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/102.0.0.0 Safari/537.36
search_query_url = https://search.brave.com/search?q=%s&source=desktop
results_container_element = div.section
result_container_element = div.snippet
result_url_element = a.result-header
result_url_property = href
result_title_element = span.snippet-header
result_title_property = text
```

### Code contribution
If you want to contribute the code you are Welcome!
