## Build and Install
### Build

* Install depenencies
 - Debian
```bash
apt-get install cmake libssl-dev libdb-dev libgumbo-dev libcurl-dev 
```

Tests dependencies (gtest):
```bash
apt-get install libgtest-dev 
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

### Build and run the tests
```bash
git clone https://github.com/krazeekermit/sniffdogsniff.git
cd sniffdogsniff
cmake -D SDS_TESTING=1 .
make
./sniffdogsniffd
```

### Init scripts
* For Linux:
An example init script can be found in /etc/init.d

* For FreeBSD:
An example init script can be found in /etc/rc.d
