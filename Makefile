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
	@echo "  retest      - run go tests, guard style"
	@echo "  build       - build binaries into bin/ directory"
	@echo "  clean       - clean up bin/ directory"
	@echo ""
	@echo "  dist        - clean build with deps and tools"
	@echo "  tools       - go get's a bunch of tools for dev"
	@echo "  deps        - pull and setup dependencies"
	@echo "  update_deps - update deps lock file"

run:
	@(export CONFIG=$$PWD/etc/imgry.conf && \
		cd ./cmd/imgry-server && fresh -w=../..)

test:
	@go test ./...

retest: test
	reflex -r "^*\.go$$" -- make test

coverage:
	@go test -cover -v ./...

build: clean
	@mkdir -p ./bin
	GOGC=off GO15VENDOREXPERIMENT=1 go build -o ./bin/imgry-server ./cmd/imgry-server

clean:
	@rm -rf ./bin

tools: dist_tools
	go get github.com/cespare/reflex
	go get github.com/pkieltyka/fresh

dist_tools:
	go get github.com/robfig/glock

deps:
	@glock sync -n github.com/pressly/imgry < Glockfile

update_deps:
	@glock save -n github.com/pressly/imgry > Glockfile

dist: clean dist_tools deps build
