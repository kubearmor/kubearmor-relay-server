CURDIR  := $(shell pwd)
VERSION := latest

.PHONY: build
build:
	cd $(CURDIR); go mod tidy
	cd $(CURDIR); go build -o kubearmor-relay-server main.go

.PHONY: run
run: $(CURDIR)/kubearmor-relay-server
	cd $(CURDIR); ./kubearmor-relay-server

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
	#cd $(CURDIR); find . -name go.sum | xargs -I {} rm -f {}
