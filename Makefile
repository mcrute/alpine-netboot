BINARY=bootstrap-server
IMAGE=docker.crute.me/netboot-server

NETBOX_HOST ?= https://netbox.example.com
HTTP_SERVER ?= http://bootstrap.example.com
VAULT_PATH ?= path/to/netbox-readonly
DEFAULT_CONFIG ?= 1

$(BINARY): $(shell find . -name '*.go')
	CGO_ENABLED=0 go build \
		-ldflags " \
			-X code.crute.us/mcrute/netboot-server/app.defaultNetboxHost=$(NETBOX_HOST) \
			-X code.crute.us/mcrute/netboot-server/app.defaultHttpServer=$(HTTP_SERVER) \
			-X code.crute.us/mcrute/netboot-server/app.defaultVaultNetboxPath=$(VAULT_PATH) \
			-X code.crute.us/mcrute/netboot-server/app.defaultNetboxConfigId=$(DEFAULT_CONFIG) \
		"  \
		-o $@

.PHONY: docker
docker: $(BINARY)
	mkdir docker; cp Dockerfile $(BINARY) docker; cd docker; \
	docker pull $(shell grep '^FROM ' Dockerfile | cut -d' ' -f2); \
	docker build --no-cache -t $(IMAGE):latest .

.PHONY: publish
publish:
	docker push $(IMAGE):latest

.PHONY: clean
clean:
	rm $(BINARY) || true
