API_IMAGE ?= nofrillz-api
AI_POSTER_IMAGE ?= nofrillz-ai-poster
SEED_IMAGE ?= nofrillz-seed
IMAGE_TAG ?= latest

API_DOCKERFILE ?= docker/api/Dockerfile
AI_POSTER_DOCKERFILE ?= docker/ai-poster/Dockerfile
SEED_DOCKERFILE ?= docker/seed/Dockerfile
DOCKER_CONTEXT ?= .
DOCKER_BUILD_FLAGS ?=

.PHONY: docker-build docker-build-api docker-build-ai-poster docker-build-seed

docker-build: docker-build-api docker-build-ai-poster docker-build-seed

docker-build-api:
	docker build $(DOCKER_BUILD_FLAGS) -f $(API_DOCKERFILE) -t $(API_IMAGE):$(IMAGE_TAG) $(DOCKER_CONTEXT)

docker-build-ai-poster:
	docker build $(DOCKER_BUILD_FLAGS) -f $(AI_POSTER_DOCKERFILE) -t $(AI_POSTER_IMAGE):$(IMAGE_TAG) $(DOCKER_CONTEXT)

docker-build-seed:
	docker build $(DOCKER_BUILD_FLAGS) -f $(SEED_DOCKERFILE) -t $(SEED_IMAGE):$(IMAGE_TAG) $(DOCKER_CONTEXT)
