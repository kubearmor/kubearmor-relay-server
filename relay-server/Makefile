CURDIR  := $(shell pwd)
VERSION := latest

.PHONY: build
build:
	cd $(CURDIR); go mod tidy
	cd $(CURDIR); go build -o kubearmor-relay-server main.go

.PHONY: run
run: $(CURDIR)/kubearmor-relay-server build
	cd $(CURDIR); ./kubearmor-relay-server

.PHONY: run-container
run-container: $(CURDIR)/kubearmor-relay-server build
	cd $(CURDIR); sudo ./kubearmor-relay-server -enableReverseLogClient -enablePolicyServer -gRPCPort=32768

.PHONY: build-image
build-image:
	docker images | grep kubearmor-relay-server | awk '{print $$3}' | xargs -I {} docker rmi {}
	cd $(CURDIR); docker build -t kubearmor/kubearmor-relay-server:$(VERSION) .

.PHONY: push-image
push-image:
	cd $(CURDIR); docker push kubearmor/kubearmor-relay-server:$(VERSION)

.PHONY: clean
clean:
	cd $(CURDIR); sudo rm -f kubearmor-relay-server

.PHONY: gofmt
gofmt:
	cd $(CURDIR); gofmt -s -d $(shell find . -type f -name '*.go' -print)

.PHONY: golint
golint:
ifeq (, $(shell which golint))
	@{ \
	set -e ;\
	GOLINT_TMP_DIR=$$(mktemp -d) ;\
	cd $$GOLINT_TMP_DIR ;\
	go mod init tmp ;\
	go get -u golang.org/x/lint/golint ;\
	rm -rf $$GOLINT_TMP_DIR ;\
	}
endif
	cd $(CURDIR); golint ./...

.PHONY: gosec
gosec:
ifeq (, $(shell which gosec))
	@{ \
	set -e ;\
	GOSEC_TMP_DIR=$$(mktemp -d) ;\
	cd $$GOSEC_TMP_DIR ;\
	go mod init tmp ;\
	go get -u github.com/securego/gosec/v2/cmd/gosec ;\
	rm -rf $$GOSEC_TMP_DIR ;\
	}
endif
	cd $(CURDIR); gosec ./...
