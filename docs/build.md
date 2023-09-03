## Build and Install
### Build

You need to setup go development environment first [see here](https://go.dev/doc/install).

This is how to build in linux and unix Operating Systems:

```bash
cd <your goroot>/src/
mkdir gitlab.com && cd gitlab.com
git clone https://gitlab.com/c3rzthefrog/sniffdogsniff.git
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