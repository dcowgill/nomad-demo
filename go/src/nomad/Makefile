all: install

makefile_dir := $(strip $(dir $(realpath $(lastword $(MAKEFILE_LIST)))))
export GOPATH := $(makefile_dir)/../..

GO := /usr/local/go/bin/go

buildinfo_pkg := nomad/buildinfo
git_rev := $(shell git rev-parse HEAD)
now := $(shell date +%Y-%m-%dT%H:%M:%S)
ldflags := -X $(buildinfo_pkg).buildSHA=$(git_rev) -X $(buildinfo_pkg).buildDate=$(now)

install:
	$(GO) install -ldflags '$(ldflags)' nomad/cmd/demo_server
	$(GO) install -ldflags '$(ldflags)' nomad/cmd/edge_server
