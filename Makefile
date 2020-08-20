.PHONY: update master release setup update_master update_release build clean version

setup:
	git config --global --add url."git@gitlab.com:".insteadOf "https://gitlab.com/"

version:
	go run main.go generate
	sed -i.bak 's/package\ cmd/package\ globals/g' version_vars.go
	mv version_vars.go globals/version_vars.go

clean:
	rm -rf vendor/
	go mod vendor

update:
	-GOFLAGS="" go get -u all

build:
	go build ./...
	go mod tidy

update_release:
	GOFLAGS="" go get -u gitlab.com/elixxir/primitives@release
	GOFLAGS="" go get -u gitlab.com/elixxir/crypto@Optimus/e2e
	GOFLAGS="" go get -u gitlab.com/xx_network/crypto@release
	GOFLAGS="" go get -u gitlab.com/elixxir/comms@release
	GOFLAGS="" go get -u gitlab.com/xx_network/comms@release
	GOFLAGS="" go get -u gitlab.com/xx_network/primitives@release

update_master:
	GOFLAGS="" go get -u gitlab.com/elixxir/primitives@master
	GOFLAGS="" go get -u gitlab.com/elixxir/crypto@master
	GOFLAGS="" go get -u gitlab.com/xx_network/crypto@master
	GOFLAGS="" go get -u gitlab.com/elixxir/comms@master
	GOFLAGS="" go get -u gitlab.com/xx_network/comms@master
	GOFLAGS="" go get -u gitlab.com/xx_network/primitives@master

master: clean update_master build version

release: clean update_release build version
