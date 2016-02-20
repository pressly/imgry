.PHONY: help run test retest coverage build clean tools dist_tools deps update_deps dist

all:
	@echo "*******************************"
	@echo "** Pressly Reeler build tool **"
	@echo "*******************************"
	@echo "make <cmd>"
	@echo ""
	@echo "commands:"
	@echo "  run         - run in dev mode"
	@echo "  test        - run go tests"
	@echo "  build       - build binaries into bin/ directory"
	@echo "  clean       - clean up bin/ directory"
	@echo ""
	@echo "  dist        - clean build with deps and tools"
	@echo "  tools       - go get's a bunch of tools for dev"

##
## Tools
##
tools:
	go get github.com/pkieltyka/fresh
	go get -u github.com/kardianos/govendor


##
## Development
##

run:
	@(export CONFIG=$$PWD/etc/imgry.conf && \
		cd ./cmd/imgry-server && fresh -w=../..)

test:
	@GOGC=off go test $$(GO15VENDOREXPERIMENT=1 go list ./... | grep -v '/vendor/')

dist-test:
	@GO15VENDOREXPERIMENT=1 $(MAKE) test



##
## Building
##
dist: clean
	GO15VENDOREXPERIMENT=1 $(MAKE) build

build:
	@mkdir -p ./bin
	GOGC=off go build -i -o ./bin/imgry-server ./cmd/imgry-server

clean:
	@rm -rf $$GOPATH/pkg/*/github.com/pressly/imgry{,.*}
	@rm -rf ./bin


##
## Dependency mgmt
##
vendor-list:
	@govendor list

vendor-update:
	@govendor update +external
