GOPATH ?= $(shell go env GOPATH)
SPDX_DATA_VERSION ?= 3.0

bindata.go: licenses.tar $(GOPATH)/bin/go-bindata
	$(GOPATH)/bin/go-bindata -pkg ld licenses.tar
	rm licenses.tar

licenses.tar: license-list-data.tar.gz
	tar -xf license-list-data.tar.gz license-list-data-$(SPDX_DATA_VERSION)/text
	tar -cf licenses.tar -C license-list-data-$(SPDX_DATA_VERSION)/text .
	rm -rf license-list-data-$(SPDX_DATA_VERSION)
	rm license-list-data.tar.gz

license-list-data.tar.gz:
	curl -SLk -o license-list-data.tar.gz https://github.com/spdx/license-list-data/archive/v$(SPDX_DATA_VERSION).tar.gz

$(GOPATH)/bin/go-bindata:
	go get -v github.com/jteeuwen/go-bindata/...
