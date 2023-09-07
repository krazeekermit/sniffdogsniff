## Build and Install
### Build

* Install Go
Download Go here [see here](https://go.dev/doc/install).

On FreeBSD you can install Go from the packages repos or from ports
```bash
pkg install lang/go
```
or
```bash
cd /usr/ports/lang/go/ && make install clean
```

* Build
```bash
cd <your goroot>/src/
mkdir github.com && cd github.com
git clone https://github.com/krazeekermit/sniffdogsniff.git
cd sniffdogsniff
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