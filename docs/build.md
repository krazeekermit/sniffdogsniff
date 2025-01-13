## Build and Install
### Build

* Install depenencies
 - Debian
```bash
apt-get install cmake libssl-dev libdb-dev libgumbo-dev libcurl-dev 
```
 - FreeBSD
```bash
pkg install cmake openssl db5 gumbo curl
```

* Build
```bash
git clone https://github.com/krazeekermit/sniffdogsniff.git
cd sniffdogsniff
cmake .
make
```

### Install
```bash
make install
```

### Init scripts
* For Linux:
An example init script can be found in /etc/init.d

* For FreeBSD:
An example init script can be found in /etc/rc.d
