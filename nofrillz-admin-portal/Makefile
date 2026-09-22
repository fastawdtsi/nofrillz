PORTAL_IMAGE ?= nofrillz-admin-portal
IMAGE_TAG ?= latest

PORTAL_DOCKERFILE ?= Dockerfile
DOCKER_CONTEXT ?= .
DOCKER_BUILD_FLAGS ?=
DOCKER_RUN_FLAGS ?=
PORTAL_PORT ?= 8080

NPM ?= npm

.PHONY: npm-install npm-dev npm-build npm-preview docker-build docker-run

npm-install:
	$(NPM) install

npm-dev:
	$(NPM) run dev

npm-build:
	$(NPM) run build

npm-preview:
	$(NPM) run preview

docker-build:
	docker build $(DOCKER_BUILD_FLAGS) -f $(PORTAL_DOCKERFILE) -t $(PORTAL_IMAGE):$(IMAGE_TAG) $(DOCKER_CONTEXT)

docker-run:
	docker run $(DOCKER_RUN_FLAGS) --rm -p $(PORTAL_PORT):8080 $(PORTAL_IMAGE):$(IMAGE_TAG)
