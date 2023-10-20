#!/bin/bash

THIS_SCRIPT_PATH=`readlink -f $0`;
THIS_SCRIPT_DIRECTORY=`dirname $0`;

GO_BIN=$GOPATH/bin;
BUILD_DIR=$THIS_SCRIPT_DIRECTORY/build/;

boldecho(){
    echo -e "$(tput bold)$1$(tput sgr0)";
}

build() {
    boldecho " -> Building SniffDogSniff";
    if [[ ! -d $BUILD_DIR ]]
    then
        mkdir $BUILD_DIR;
    fi
    go build -v -o $BUILD_DIR;
    if [[ $? -ne 0 ]]
    then
      echo " [ERR] Compile errors"
      exit $?
    fi
}

run() {
    build;
    boldecho " -> Running SniffDogSniff";
    cp $THIS_SCRIPT_DIRECTORY/config.ini.sample $BUILD_DIR/config.ini;
    echo "$BUILD_DIR/config.ini"
    $BUILD_DIR/sniffdogsniff -c $BUILD_DIR/config.ini --log-level DEBUG;
}

for arg in $@
do
    case $arg in
    "run")
    run;
    ;;
    "build")
    build;
    ;;
    esac
done

boldecho " <- Done";

