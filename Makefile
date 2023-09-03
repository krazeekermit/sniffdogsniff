version := v0.1_alpha
build_dir := build/${version}

bin_dir := /usr/local/bin
etc_dir := /usr/local/etc

GO := go
GO_FLAGS := -v

sdsbuild:
	mkdir -p ${build_dir}
	${GO} build ${GO_FLAGS} -o ${build_dir}
	cp config.ini.sample ${build_dir}/sniffdogsniff.ini

all: sdsbuild

test:
	${GO} test ${GO_FLAGS} ./core
	${GO} test ${GO_FLAGS} ./kademlia

install: 
	install -d ${bin_dir}
	install -m 600 ${build_dir}/sniffdogsniff ${bin_dir}
	install -d ${etc_dir}
	install -m 600 ${build_dir}/config.ini ${etc_dir}

clean:
	rm -r ${build_dir}
	