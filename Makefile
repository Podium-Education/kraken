BUILD_COMMIT := $(shell git describe --tags --always --dirty --all --match=v*)
BUILD_DATE := $(shell date -u +%b-%d-%Y,%T-UTC)
BUILD_SEMVER := $(shell cat .SEMVER)

.PHONY: all build

build:
	go build .
	mv kraken /usr/local/bin/kraken
