## Build

You need to setup go development environment first [see here](https://go.dev/doc/install).

This is how to build in linux and unix Operating Systems:

```bash
cd <your goroot>/src/
mkdir gitlab.com && cd gitlab.com
git clone https://gitlab.com/c3rzthefrog/sniffdogsniff.git
cd sniffdogsniff
./build.sh build # This simple script avoid creation of untracked files
```