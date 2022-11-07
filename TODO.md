* priority: HIGH -> fix rpc server to work with non-blocking sockets
* database v2 (+ tags: list)
* search on database improvements (search by tag)
* packaging (pyinstaller) 
  
* multiple simultaneous sync request test && evaluate adding a request queue
* test fix
* review sync protocol (is json-rpc the best choice?; MSGPACK?; evaluate more bandwidth saving alternative especially because we want to work mainly under Tor/I2p networks that are very slow)
